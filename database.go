package main

import (
	"github.com/Pie-Messaging/core/pie"
	"github.com/Pie-Messaging/core/pie/routing"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gLog "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"path/filepath"
	"time"
)

type User struct {
	ID        *pie.ID      `gorm:"primaryKey"`
	Name      string       `gorm:"default:''"`
	Email     string       `gorm:"default:''"`
	Bio       string       `gorm:"default:''"`
	HasAvatar bool         `gorm:"default:false"`
	CertDER   []byte       `gorm:""`
	Addr      routing.Addr `gorm:"type:bytes"`
}

func openDB() {
	var err error
	level := logger.GetLevel()
	db, err = gorm.Open(sqlite.Open(filepath.Join(configDir, "main.db")), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger: gLog.New(
			logger,
			gLog.Config{
				SlowThreshold: 200 * time.Millisecond,
				LogLevel:      gLog.LogLevel(pie.MinInt(pie.MaxInt(int(level), int(gLog.Silent)), int(gLog.Info))),
				Colorful:      true,
			},
		),
	})
	if err != nil {
		logger.Fatalln("Failed to connect database:", err)
	}
	err = db.AutoMigrate(&PeerTracker{}, &User{})
	if err != nil {
		logger.Fatalln("Failed to create/alter database tables:", err)
	}
}
