package database

import (
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewConnection(dsn string) (*gorm.DB, error) {
	log.Printf("Connecting to database with DSN: %s", dsn)
	var db *gorm.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}

		log.Println("Database not ready, retrying...")
		time.Sleep(3 * time.Second)
	}

	return db, nil
}
