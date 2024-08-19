package routes

import (
	"errors"
	"net/http"
	"persona/database"
	"persona/utils"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitChat(c *gin.Context) (*gorm.DB, utils.Akun, utils.Karakter, utils.KarakterChat, error) {
	dbRaw, _ := c.Get("db")
	db := dbRaw.(*gorm.DB)

	akunRaw, _ := c.Get("akun")
	akun := akunRaw.(utils.Akun)

	var karakter utils.Karakter
	db.Where("ID = ?", c.Param("idchat")).First(&karakter)

	if karakter.ID == 0 {
		return db, akun, karakter, utils.KarakterChat{}, errors.New("tidak ada karakter")
	}

	karakterChat := utils.KarakterChat{}
	db.Where("karakter_id = ? AND pechat_id = ?", karakter.ID, akun.ID).Preload("History").First(&karakterChat)

	return db, akun, karakter, karakterChat, nil
}

func DapatinAkun() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		db := database.GetDatabase()
		session := sessions.Default(ctx)
		user := session.Get("user")

		if user == nil {
			ctx.Redirect(http.StatusFound, "/login")
			return
		}

		urlPath := strings.Split(ctx.Request.URL.Path, "/")
		joinsString := []string{}

		if (urlPath[0] == "chat" && urlPath[2] != "hapuspesan") || urlPath[0] == "personalitas" {
			joinsString = append(joinsString, "Personalitas")
		}

		akun := utils.DapatinAkun(db, session, &joinsString)

		ctx.Set("db", db)
		ctx.Set("akun", akun)
	}
}

func CheckAutentikasi(status string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
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
