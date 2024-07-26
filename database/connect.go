package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var database *gorm.DB

func Connect() {
	conectDB := "postgresql://rapithon:gmCePyjssc9j9I5hw29ymg@per-chat-7248.6xw.aws-ap-southeast-1.cockroachlabs.cloud:26257/defaultdb?sslmode=verify-full" // os.Getenv("DATABASE_URI")
	db, _ := gorm.Open(postgres.Open(conectDB), &gorm.Config{})
	fmt.Println("Database Connected")

	database = db
	config, _ := database.DB()
	config.SetMaxIdleConns(10)
	config.SetMaxOpenConns(100)
	config.SetConnMaxLifetime(time.Hour)

	database.AutoMigrate(&Akun{})
	database.AutoMigrate(&Karakter{})
	database.AutoMigrate(&KarakaterChat{})
	database.AutoMigrate(&IsiChat{})

	fmt.Println("Migrations Finished")
}

func CloseCon() {
	config, _ := database.DB()
	config.Close()
}

func GetDatabase() *gorm.DB {
	return database
}
