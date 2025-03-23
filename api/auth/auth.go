package auth

import (
	"persona/database/models"

	"github.com/gin-contrib/sessions"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func DapatinAkun(db *gorm.DB, session sessions.Session, joins *[]string) models.Akun {
	user := session.Get("user")

	if user == nil {
		return models.Akun{}
	}

	var akun models.Akun
	TempDB := db

	if joins != nil {
		for _, v := range *joins {
			TempDB = TempDB.Joins(v)
		}
	}

	TempDB.First(&akun, user)

	return akun
}
