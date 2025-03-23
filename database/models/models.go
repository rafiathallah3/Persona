package models

import (
	"fmt"
	"strings"
	"time"
)

type Akun struct {
	ID                uint64         `json:"id" gorm:"primaryKey;autoIncrement:true"` //gorm:"type:uuid;default:uuid_generate_v4()"
	Username          string         `json:"username" gorm:"size:20"`
	Email             string         `json:"email" gorm:"unique;size:60"`
	Password          string         `json:"password"`
	ImageURL          string         `json:"image_url" gorm:"default:'/assets/no-users.png'"`
	PembuatanKarakter []Karakter     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignkey:AkunID"`
	ChatKarakter      []KarakterChat `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignkey:PechatID"`
	ListPersonalitas  []Personalitas `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignkey:AkunID"`
	PersonalitasID    *uint64        `gorm:"index"`
	Personalitas      Personalitas   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:PersonalitasID"`
	CreatedAt         time.Time
}

type Personalitas struct {
	ID           uint64 `json:"id" gorm:"primaryKey"`
	Nama         string `json:"nama" gorm:"size:40"`
	Personalitas string `json:"personalitas" gorm:"size:150"`
	AkunID       uint64 `gorm:"index"`
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

type Karakter struct {
	ID                uint64 `json:"id" gorm:"primaryKey"`
	Nama              string `json:"nama"`
	NamaLain          string `json:"namalain"`
	Deskripsi         string `json:"deskripsi"`
	Personalitas      string `json:"personalitas"`
	Kategori          string `json:"kategori"`
	Chat              string `json:"chat"`
	Gambar            string
	CreatedAt         time.Time
	SemuaKarakterChat []KarakterChat `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignkey:KarakterID"`
	AkunID            uint64         `gorm:"index"`
	Akun              Akun
}

func (karakter Karakter) RenderPersonalitas(username string) string {
	return fmt.Sprintf("Your name is %s, You are currently talking with %s", karakter.Nama, karakter.Nama) + strings.ReplaceAll(strings.ReplaceAll(karakter.Personalitas, "{{char}}", karakter.Nama), "{{user}}", username)
}

func (karakter Karakter) RenderChat(username string) string {
	return strings.ReplaceAll(strings.ReplaceAll(karakter.Chat, "{{char}}", karakter.Nama), "{{user}}", username)
}

type KarakterChat struct {
	ID         uint64    `gorm:"primaryKey"`
	History    []IsiChat `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignkey:RoomChatID"`
	KarakterID uint64    `gorm:"index"`
	Karakter   Karakter
	PechatID   uint64 `gorm:"index"`
	Pechat     Akun
	CreatedAt  time.Time
}

type IsiChat struct {
	ID          uint64 `gorm:"primaryKey"`
	Chat        string `json:"chat" gorm:"size:1000;not null"`
	Role        string `json:"role" gorm:"size:5;not null"`
	Posisi      uint8  `gorm:"defualt:1"`
	CreatedAt   time.Time
	RoomChatID  uint64
	DariPecatID uint64
}

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

type ListChat struct {
	IDChat       uint64
	IDKarakter   uint64
	ChatTerakhir string
	Nama         string
	Gambar       string
	Tag          string
	CreatedAt    time.Time
}

type DataInitChat struct {
	Karakter     Karakter
	KarakterChat KarakterChat
	PostChat     PostChat
}
