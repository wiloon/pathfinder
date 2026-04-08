package storage

import (
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dsn string) {
	var err error
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := DB.AutoMigrate(&User{}, &UserProfile{}, &Goal{}, &GoalAttachment{}, &DailyPlan{}, &Task{}, &Event{}, &EventAttachment{}, &CheckIn{}); err != nil {
		log.Fatalf("failed to auto-migrate database schema: %v", err)
	}
}
