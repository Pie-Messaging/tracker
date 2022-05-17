package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Pie-Messaging/core/pie"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

func main() {
	pie.SetLogOutput("")
	logger = NewLogger()
	logger.Infoln("Starting Pie")

	processPIDFile()

	logLevel := flag.String("l", "debug", "Log level")
	runInBackground := flag.Bool("run-bg", true, "Run in background")
	flag.StringVar(&configDir, "config", "", "Dir to config files")
	peerTrackerStrs := flag.String("tracker", "", "Space separated list of peer tracker addresses")
	flag.Parse()

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		logger.Fatalln("Invalid log level:", err)
	}
	logger.SetLevel(level)

	var logFile *os.File

	if *runInBackground {
		logFile = setLogFile()
		defer func() {
			err := logFile.Sync()
			if err != nil {
				logger.Warnln("Failed to close log file:", err)
			}
		}()
	}

	if configDir == "" {
		defaultConfigDir, err := os.UserConfigDir()
		if err != nil {
			logger.Fatalln("Failed to get user config dir:", err)
		}
		configDir = filepath.Join(defaultConfigDir, "pie", "tracker")
	}
	filesDir = filepath.Join(configDir, "files")
	err = os.MkdirAll(filesDir, 0774)
	if err != nil {
		logger.Fatalln("Failed to create files dir:", err)
	}
	configFilePath := filepath.Join(configDir, "config.yaml")

	certFilePath := path.Join(configDir, "id.crt")
	keyFilePath := path.Join(configDir, "id.key")
	var certificate *tls.Certificate

	_, err = os.Stat(configDir)
	if err == nil {
		certificate = readConfig(configFilePath, certFilePath, keyFilePath)
	} else if os.IsNotExist(err) {
		// First time running
		certificate = createConfig(certFilePath, keyFilePath)
	} else {
		logger.Fatalln("Failed to read stat of config dir:", err)
	}

	addrList := make([]string, 0, 1)
	interfaceAddr, err := net.InterfaceAddrs()
	if err != nil {
		logger.Fatalln("Failed to get interface addresses:", err)
	}
	for _, addr := range interfaceAddr {
		if ip, ok := addr.(*net.IPNet); ok && ip.IP.IsGlobalUnicast() && !ip.IP.IsPrivate() {
			addrList = append(addrList, ip.IP.String())
		}
	}
	if len(addrList) == 0 {
		logger.Fatalln("No global IP address found")
	}

	openDB()

	ctx, shutdown = context.WithCancel(context.Background())
	listenOSSignal()

	dbTrackers := make([]*PeerTracker, 0, pie.KSize)
	if *peerTrackerStrs == "" {
		db.Model(&PeerTracker{}).Find(&dbTrackers)
	} else {
		for _, peerTrackerStr := range strings.Split(*peerTrackerStrs, " ") {
			addr, err := json.Marshal(peerTrackerStr)
			if err != nil {
				logger.Fatalln("Failed to marshal peer tracker address from arguments:", err)
			}
			dbTracker := &PeerTracker{Addr: string(addr)}
			dbTrackers = append(dbTrackers, dbTracker)
		}
	}
	thisTracker = &Tracker{
		listenAddr: fmt.Sprintf(":%d", config.Port),
		addr:       addrList,
	}
	thisTracker.init(certificate, dbTrackers)
	logger.Infoln("My ID:", hex.EncodeToString(thisTracker.id.Bytes()))

	waitGroup = &sync.WaitGroup{}
	startupChan := make(chan int)
	go thisTracker.run(startupChan)

	port := <-startupChan
	for _, addr := range addrList {
		logger.Printf("Listening on [%s]:%d", addr, port)
	}
	config.Port = port
	writeConfig(configFilePath)

	logger.Infoln("Started Pie")
	if *runInBackground {
		logger.SetOutput(logFile)
		logger.SetOutput(logFile)
	}

	<-ctx.Done()
	waitGroup.Wait()
}

func listenOSSignal() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		shutdown()
	}()
}

func processPIDFile() {
	pidFilePath := filepath.Join(os.TempDir(), "pie", "tracker.pid")
	pidBytes, err := os.ReadFile(pidFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		logger.Fatalln("Failed to read pid file:", err)
	}
	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil {
		logger.Fatalln("Failed to parse pid file:", err)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		logger.Fatalln("Failed to find process:", err)
	}
	if err := process.Signal(syscall.Signal(0)); err != nil && !errors.Is(err, os.ErrProcessDone) {
		logger.Fatalln("Another instance of Pie Tracker is already running")
	}
	err = os.WriteFile(pidFilePath, []byte(strconv.Itoa(os.Getpid())), 0o644)
	if err != nil {
		logger.Fatalln("Failed to write pid file:", err)
	}
}
