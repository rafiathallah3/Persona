package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/google/generative-ai-go/genai"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type DataHistoryChat struct {
	ID    uint64
	Chat  string
	Role  string
	Waktu time.Time
}

type PostChat struct {
	KarakterID string `json:"karakterID"`
	ChatID     string `json:"chatID"`
	PesanID    string `json:"pesanID"`
	Chat       string `json:"chat"`
}

type DataInitChat struct {
	Karakter     Karakter
	KarakterChat KarakterChat
	PostChat     PostChat
}

type ListChat struct {
	IDChat       uint64
	IDKarakter   uint64
	ChatTerakhir string
	Nama         string
	Gambar       string
	Tag          string
	CreatedAt    time.Time
}

var Kategori = []string{"Anime", "Comic", "Movie", "Girl", "Boy"}

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

func (personalitas Personalitas) RenderPersonalitas(username string) string {
	return fmt.Sprintf("Your name is %s, ", personalitas.Nama) + strings.ReplaceAll(strings.ReplaceAll(personalitas.Personalitas, "{{char}}", personalitas.Nama), "{{user}}", username)
}

func (karakter Karakter) RenderPersonalitas(username string) string {
	return fmt.Sprintf("Your name is %s, You are currently talking with %s", karakter.Nama, karakter.Nama) + strings.ReplaceAll(strings.ReplaceAll(karakter.Personalitas, "{{char}}", karakter.Nama), "{{user}}", username)
}

func (karakter Karakter) RenderChat(username string) string {
	return strings.ReplaceAll(strings.ReplaceAll(karakter.Chat, "{{char}}", karakter.Nama), "{{user}}", username)
}

// Why not use this function to update the position? Because if the records are created at the same time. The length of the table would just return the same value
// func (isiChat *IsiChat) AfterCreate(tx *gorm.DB) (err error) {
// 	var listChatRoom []IsiChat

// 	tx.Model(IsiChat{ID: isiChat.ID}).Find(&listChatRoom)

// 	tx.Statement.Update("posisi", len(listChatRoom)+1)

// 	return
// }

func DapatinHistoryKarakter(karakterChat KarakterChat) ([]*genai.Content, []DataHistoryChat) {
	genAIHistoryChat := []*genai.Content{}
	dataHistoryChat := []DataHistoryChat{}

	sort.Slice(karakterChat.History, func(i, j int) bool {
		return karakterChat.History[i].Posisi < karakterChat.History[j].Posisi
	})

	for _, v := range karakterChat.History {
		genAIHistoryChat = append(genAIHistoryChat, &genai.Content{
			Role:  v.Role,
			Parts: []genai.Part{genai.Text(v.Chat)},
		})

		dataHistoryChat = append(dataHistoryChat, DataHistoryChat{
			ID:    v.ID,
			Chat:  v.Chat,
			Role:  v.Role,
			Waktu: v.CreatedAt,
		})
	}

	return genAIHistoryChat, dataHistoryChat
}

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
