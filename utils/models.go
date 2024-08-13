package utils

import (
	"time"
)

type Akun struct {
	ID                uint64         `json:"id" gorm:"primaryKey;autoIncrement:true"` //gorm:"type:uuid;default:uuid_generate_v4()"
	Username          string         `json:"username" gorm:"size:20"`
	Email             string         `json:"email" gorm:"unique;size:60"`
	Password          string         `json:"password"`
	ImageURL          string         `json:"image_url" gorm:"default:'/assets/no-users.png'"`
	PembuatanKarakter []Karakter     `gorm:"constraint:OnUpdate:CASCADE;foreignkey:AkunID"`
	ChatKarakter      []KarakterChat `gorm:"constraint:OnUpdate:CASCADE;foreignkey:PechatID"`
	ListPersonalitas  []Personalitas `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignkey:AkunID"`
	PersonalitasID    uint64         `gorm:"index"`
	Personalitas      Personalitas   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:PersonalitasID"`
	CreatedAt         time.Time
}

type Personalitas struct {
	ID           uint64 `json:"id" gorm:"primaryKey"`
	Nama         string `json:"nama" gorm:"size:40"`
	Personalitas string `json:"personalitas" gorm:"size:150"`
	AkunID       uint64 `gorm:"index"`
}

type Karakter struct {
	ID                uint64 `json:"id" gorm:"primaryKey"`
	Nama              string `json:"nama"`
	NamaLain          string `json:"namalain"`
	Personalitas      string `json:"personalitas"`
	Kategori          string `json:"kategori"`
	Chat              string `json:"chat"`
	Gambar            string
	CreatedAt         time.Time
	SemuaKarakterChat []KarakterChat `gorm:"constraint:OnUpdate:CASCADE;foreignkey:KarakterID"`
	AkunID            uint64         `gorm:"index"`
	Akun              Akun
}

type KarakterChat struct {
	ID         uint64    `gorm:"primaryKey"`
	History    []IsiChat `gorm:"constraint:OnUpdate:CASCADE;foreignkey:ID"`
	KarakterID int       `gorm:"index"`
	PechatID   uint64    `gorm:"index"`
}

type IsiChat struct {
	ID             uint64 `gorm:"primaryKey"`
	Chat           string `json:"chat" gorm:"size:150;not null"`
	Role           string `json:"role" gorm:"size:5;not null"`
	CreatedAt      time.Time
	DariKarakterID uint64
}
