package main

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

type Logger struct {
	*log.Logger
}

func (l *Logger) Shutdown(args ...any) {
	l.Logger.Errorln(args...)
	l.Logger.Warnln("Shutting down")
	shutdown()
}

func NewLogger() *Logger {
	l := log.New()
	l.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	return &Logger{l}
}

func setLogFile() *os.File {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		logger.Fatalln("Failed to get user cache dir:", err)
	}
	cacheDir = filepath.Join(cacheDir, "pie", "tracker")
	if err = os.MkdirAll(cacheDir, 0774); err != nil {
		logger.Fatalln("Failed to create cache dir:", err)
	}
	logFile, err := os.OpenFile(filepath.Join(cacheDir, "tracker.log"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		logger.Fatalln("Failed to open log file:", err)
	}
	_, err = logFile.WriteString("Pie Tracker Debug Log\nThis file will be cleared on every start\n\n")
	if err != nil {
		logger.Errorln("Failed to write to log file:", err)
	}
	return logFile
}
