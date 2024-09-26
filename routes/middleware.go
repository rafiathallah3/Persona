package routes

import (
	"errors"
	"fmt"
	"net/http"
	"persona/database"
	"persona/utils"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitAkunDB(c *gin.Context) (*gorm.DB, utils.Akun) {
	dbRaw, _ := c.Get("db")
	db := dbRaw.(*gorm.DB)

	akunRaw, _ := c.Get("akun")
	akun := akunRaw.(utils.Akun)

	return db, akun
}

func InitChat(c *gin.Context) (utils.DataInitChat, error) {
	dataInitChat := utils.DataInitChat{}

	db, akun := InitAkunDB(c)

	var dataPost utils.PostChat
	idKarakter := strings.ReplaceAll(c.Param("idkarakter"), "/", "")
	idChat := strings.ReplaceAll(c.Param("idchat"), "/", "")

	if c.Request.Method == "POST" {
		postKarakterID := c.PostForm("karakterID")

		if postKarakterID == "" {
			if err := c.Bind(&dataPost); err != nil {
				return dataInitChat, errors.New("paramater missing")
			}
		} else {
			dataPost = utils.PostChat{
				KarakterID: c.PostForm("karakterID"),
				ChatID:     c.PostForm("chatID"),
			}
		}

		idKarakter = dataPost.KarakterID
		dataInitChat.PostChat = dataPost
		idChat = dataPost.ChatID

		fmt.Println("POST!!")
		fmt.Println(dataPost.KarakterID)
		fmt.Println(dataPost.Chat)
	}

	var karakter utils.Karakter
	db.Where("ID = ?", idKarakter).First(&karakter)

	dataInitChat.Karakter = karakter

	if karakter.ID == 0 {
		return dataInitChat, errors.New("tidak ada karakter")
	}

	karakterChat := utils.KarakterChat{}
	if idChat != "" {
		db.Where("id = ? AND pechat_id = ?", idChat, akun.ID).Preload("History").First(&karakterChat)
	}

	if karakterChat.ID == 0 && idChat == "" {
		db.Where("karakter_id = ? AND pechat_id = ?", karakter.ID, akun.ID).Preload("History").First(&karakterChat)
	}

	dataInitChat.KarakterChat = karakterChat

	return dataInitChat, nil
}

var ContohPathPersonalitas = []string{"/api/chat/buatchat", "/api/chat/ulangipesan", "/api/chat/sarankalimat", "/api/chat/", "/personalitas"}

func DapatinAkun() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fmt.Println("dapatinAkun")

		db := database.GetDatabase()
		session := sessions.Default(ctx)
		user := session.Get("user")

		if user == nil {
			ctx.Redirect(http.StatusFound, "/login")
			return
		}

		splitPath := strings.Split(ctx.Request.URL.Path, "/")
		joinsString := []string{}

		if utils.StringDiSlice(ctx.Request.URL.Path, ContohPathPersonalitas) || splitPath[1] == "chat" {
			joinsString = append(joinsString, "Personalitas")
		}

		akun := utils.DapatinAkun(db, session, &joinsString)

		ctx.Set("db", db)
		ctx.Set("akun", akun)
	}
}

func CheckAutentikasi(status string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fmt.Println("AAA")
		session := sessions.Default(ctx)
		user := session.Get("user")

		if status == "akses" {
			if user == nil {
				ctx.Redirect(http.StatusFound, "/login")
				return
			}

			// db := database.GetDatabase()
			akunRaw, _ := ctx.Get("akun")
			akun := akunRaw.(utils.Akun)
			// akun := utils.DapatinAkun(db, session, nil)

			// fmt.Println("ID: " + akun.ID.String())
			if akun.ID == 0 {
				session.Delete("user")
				session.Save()

				ctx.Redirect(http.StatusFound, "/login")
				return
			}

			session.Save()
			return
		}

		if status == "login" && user != nil {
			db := database.GetDatabase()
			akun := utils.DapatinAkun(db, session, nil)

			// fmt.Println("ID: " + akun.ID.String())
			if akun.ID == 0 {
				session.Delete("user")
				session.Save()
				ctx.Redirect(http.StatusFound, "/login")
				return
			}

			ctx.Redirect(http.StatusFound, "/")
			return
		}
	}
}
