package main

import (
	"context"
	"gorm.io/gorm"
	"sync"
)

var (
	configDir string
	filesDir  string
)

var (
	thisTracker *Tracker
	config      *Config
	db          *gorm.DB
	logger      *Logger
	waitGroup   *sync.WaitGroup
	ctx         context.Context
	shutdown    context.CancelFunc
)
