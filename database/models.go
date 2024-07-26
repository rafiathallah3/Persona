package database

import (
	"time"

	"gorm.io/gorm"
)

type Akun struct {
	gorm.Model
	ID                uint            `json:"id" gorm:"primaryKey"`
	Username          string          `json:"username"`
	Email             string          `json:"email" gorm:"unique"`
	Password          string          `json:"password"`
	ImageURL          string          `json:"image_url" gorm:"default:'/assets/no-users.png'"`
	PembuatanKarakter []Karakter      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignkey:ID"`
	ChatKarakter      []KarakaterChat `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignkey:ID"`
	CreatedAt         time.Time
}

type Karakter struct {
	gorm.Model
	ID                uint   `json:"id" gorm:"primaryKey"`
	Nama              string `json:"nama"`
	NamaLain          string `json:"namalain"`
	Personalitas      string `json:"personalitas"`
	Kategori          string `json:"kategori"`
	Gambar            string `json:"gambar"`
	CreatedAt         time.Time
	SemuaKarakterChat []KarakaterChat `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignkey:ID"`
	PembuatID         uint
	Pembuat           Akun
}

type KarakaterChat struct {
	gorm.Model
	ID       uint      `gorm:"primaryKey"`
	History  []IsiChat `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignkey:ID"`
	ChatID   uint
	Chat     Karakter
	PechatID uint
	Pechat   Akun
}

type IsiChat struct {
	ID         uint   `gorm:"primaryKey"`
	Chat       string `json:"chat" gorm:"size:150;not null"`
	Role       string `json:"role" gorm:"size:5;not null"`
	CreatedAt  time.Time
	DariChatID uint
	DariChat   KarakaterChat
}
