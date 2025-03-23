package utils

import (
	"encoding/json"
	"os"

	"github.com/joho/godotenv"
)

var Kategori = []string{"Anime", "Comic", "Movie", "Girl", "Boy"}

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

// Why not use this function to update the position? Because if the records are created at the same time. The length of the table would just return the same value
// func (isiChat *IsiChat) AfterCreate(tx *gorm.DB) (err error) {
// 	var listChatRoom []IsiChat

// 	tx.Model(IsiChat{ID: isiChat.ID}).Find(&listChatRoom)

// 	tx.Statement.Update("posisi", len(listChatRoom)+1)

// 	return
// }

func StringDiSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
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

func PanjangArrayKurangSatu(arr []any) int {
	return len(arr)
}
