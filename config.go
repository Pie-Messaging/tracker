package main

import (
	"crypto/tls"
	"github.com/Pie-Messaging/core/pie"
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

const (
	recvTimeout     = time.Second * 30
	recvCertTimeout = time.Second * 30
)

type Config struct {
	Port int `yaml:"port"`
}

func readConfig(configFilePath string, certFilePath string, keyFilePath string) *tls.Certificate {
	configBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		pie.Logger.Fatalln("Failed to read config.yaml:", err)
	}
	config = &Config{}
	err = yaml.Unmarshal(configBytes, config)
	if err != nil {
		pie.Logger.Fatalln("Failed to parse config.yaml:", err)
	}
	cert, err := tls.LoadX509KeyPair(certFilePath, keyFilePath)
	if err != nil {
		pie.Logger.Fatalln("Failed to load certificate:", err)
	}
	return &cert
}

func writeConfig(configFilePath string) {
	configBytes, err := yaml.Marshal(config)
	if err != nil {
		pie.Logger.Fatalln("Failed to serialize config:", err)
	}
	err = os.WriteFile(configFilePath, configBytes, 0o640)
	if err != nil {
		pie.Logger.Fatalln("Failed to write config file:", err)
	}
}

func createConfig(certFilePath string, keyFilePath string) *tls.Certificate {
	if err := os.MkdirAll(configDir, 0o774); err != nil {
		pie.Logger.Fatalln("Failed to create config dir:", err)
	}
	certificate, certPEM, keyPEM, err := pie.GenerateKeyPair()
	if err != nil {
		pie.Logger.Fatalln("Failed to generate key pair:", err)
	}
	err = os.WriteFile(certFilePath, certPEM, 0o644)
	if err != nil {
		pie.Logger.Fatalln("Failed to write certificate file:", err)
	}
	err = os.WriteFile(keyFilePath, keyPEM, 0o400)
	if err != nil {
		pie.Logger.Fatalln("Failed to write key file:", err)
	}
	return certificate
}
