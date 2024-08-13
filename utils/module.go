package utils

import (
	"encoding/json"
	"os"

	"github.com/gin-contrib/sessions"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func DapatinAkun(db *gorm.DB, session sessions.Session, joins *[]string) Akun {
	user := session.Get("user")

	if user == nil {
		return Akun{}
	}

	var akun Akun
	TempDB := db

	if joins != nil {
		for _, v := range *joins {
			TempDB = TempDB.Joins(v)
		}
	}

	TempDB.First(&akun, user)

	return akun
}

func (personalitas Personalitas) DefaultPersonalitas(NamaDefault string) Personalitas {
	if personalitas.ID == 0 {
		personalitas.Nama = NamaDefault
	}

	return personalitas
}

func DapatinEnvVariable(key string) string {
	err := godotenv.Load(".env")

	if err != nil {
		return os.Getenv(key)
	}

	return os.Getenv(key)
}

func DeepCopy(src, dst interface{}) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dst)
}
