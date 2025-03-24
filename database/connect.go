package database

import (
	"fmt"
	"persona/database/models"
	"persona/utils"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
)

var SecretKey = []byte(utils.DapatinEnvVariable("SECRETKEY"))
var db *gorm.DB
var rds *redis.Client

func Connect() {
	conectDB := utils.DapatinEnvVariable("DATABASE")
	redisDB := utils.DapatinEnvVariable("REDIS")

	db, _ = gorm.Open(postgres.Open(conectDB), &gorm.Config{})
	fmt.Println("Database Connected")

	config, _ := db.DB()
	config.SetMaxIdleConns(10)
	config.SetMaxOpenConns(100)
	config.SetConnMaxLifetime(time.Hour)

	// db.Migrator().CreateTable(&utils.IsiChat{})
	// db.Migrator().RenameColumn(&utils.IsiChat{}, "dari_karakter_id", "room_chat_id")
	db.AutoMigrate(&models.Akun{}, &models.Personalitas{}, &models.Karakter{}, &models.KarakterChat{}, &models.IsiChat{})

	opt, _ := redis.ParseURL(redisDB)
	rds = redis.NewClient(opt)

	fmt.Println("Migrations Finished")
}

func CloseCon() {
	config, _ := db.DB()
	config.Close()
}

func GetDatabase() (*gorm.DB, *redis.Client) {
	return db, rds
}
