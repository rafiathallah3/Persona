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
	// conectDB := "postgresql://rapithon:gmCePyjssc9j9I5hw29ymg@per-chat-7248.6xw.aws-ap-southeast-1.cockroachlabs.cloud:26257/defaultdb?sslmode=verify-full" // os.Getenv("DATABASE_URI")
	db, _ = gorm.Open(postgres.Open(conectDB), &gorm.Config{})
	fmt.Println("Database Connected")

	config, _ := db.DB()
	config.SetMaxIdleConns(10)
	config.SetMaxOpenConns(100)
	config.SetConnMaxLifetime(time.Hour)

	// db.Migrator().CreateTable(&utils.IsiChat{})
	// db.Migrator().RenameColumn(&utils.IsiChat{}, "dari_karakter_id", "room_chat_id")
	db.AutoMigrate(&models.Akun{}, &models.Personalitas{}, &models.Karakter{}, &models.KarakterChat{}, &models.IsiChat{})

	opt, _ := redis.ParseURL("rediss://default:ATyqAAIjcDFiNDg3M2FhYmFhNjI0NmNhOWViZGY4MmNkNDYwNTVhMXAxMA@rational-seahorse-15530.upstash.io:6379")
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
