package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"

	"strings"

	"persona/database"
	"persona/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	// "github.com/google/uuid"
)

type CHAT struct {
	Id   string `json:"id"`
	Chat string `json:"chat"`
}

type StatusPeronsalitas struct {
	Status       string `json:"status"`
	ID           string `json:"id"`
	Nama         string `json:"nama"`
	Personalitas string `json:"personalitas"`
}

var secret = []byte("Rahasia")

func PanjangArrayKurangSatu(arr []any) int {
	return len(arr)
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

			db := database.GetDatabase()
			akun := utils.DapatinAkun(db, session, nil)

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

			fmt.Println(akun.ID)

			// fmt.Println("ID: " + akun.ID.String())
			if akun.ID == 0 {
				fmt.Println("HANCUR!")
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

func main() {
	// sk-proj-pSehnOF8HPuzFkEU2NK0T3BlbkFJxrUc1Pu0PXkHruHhzjbN
	// gmCePyjssc9j9I5hw29ymg
	// postgresql://rapithon:gmCePyjssc9j9I5hw29ymg@per-chat-7248.6xw.aws-ap-southeast-1.cockroachlabs.cloud:26257/defaultdb?sslmode=verify-full

	database.Connect()

	client := utils.ClientGenAI()
	defer client.Close()

	r := gin.Default()
	r.SetFuncMap(template.FuncMap{
		"PanjangArrayKurangSatu": PanjangArrayKurangSatu,
	})
	r.Static("/assets", "./assets")
	r.LoadHTMLGlob("templates/*")
	r.StaticFile("/favicon.ico", "./assets/IconPer.png")
	r.Use(sessions.Sessions("session", cookie.NewStore(secret)))
	r.MaxMultipartMemory = 5 << 20

	r.GET("/", func(ctx *gin.Context) {
		db := database.GetDatabase()
		session := sessions.Default(ctx)

		akun := utils.DapatinAkun(db, session, nil)

		var SemuaKarakter []utils.Karakter
		db.Find(&SemuaKarakter)

		ctx.HTML(http.StatusOK, "index.html", gin.H{
			"title":    "Main",
			"akun":     akun,
			"karakter": SemuaKarakter,
		})
	})

	r.GET("/profile", func(c *gin.Context) {
		c.HTML(http.StatusOK, "profile.html", gin.H{
			"title": "Profile",
		})
	})

	r.POST("/logout", func(ctx *gin.Context) {
		session := sessions.Default(ctx)
		user := session.Get("user")

		if user == nil {
			ctx.Redirect(http.StatusFound, "/login")
			return
		}

		session.Delete("user")

		if err := session.Save(); err != nil {
			ctx.Redirect(http.StatusFound, "/login")
			return
		}

		ctx.Redirect(http.StatusFound, "/login")
	})

	redirectLoginAutentikasi := r.Group("/")
	redirectLoginAutentikasi.Use(CheckAutentikasi("akses"))
	{
		redirectLoginAutentikasi.GET("/buatkarakter/", func(ctx *gin.Context) {
			db := database.GetDatabase()
			session := sessions.Default(ctx)
			user := session.Get("user")

			var akun utils.Akun
			db.First(&akun, user)

			ctx.HTML(http.StatusOK, "karakter.html", gin.H{
				"title": "Create Character",
				"akun":  akun,
			})
		})

		redirectLoginAutentikasi.GET("/editkarakter/:idkarakter", func(ctx *gin.Context) {
			db := database.GetDatabase()
			session := sessions.Default(ctx)
			user := session.Get("user")

			var akun utils.Akun
			db.First(&akun, user)

			var karakter utils.Karakter
			db.Where("ID = ?", ctx.Param("idkarakter")).First(&karakter)

			if karakter.AkunID != akun.ID {
				ctx.HTML(http.StatusNotFound, "error.html", gin.H{
					"title": "Error",
					"akun":  akun,
					"kode":  404,
					"isi":   "Character not found",
				})
				return
			}

			ctx.HTML(http.StatusOK, "karakter.html", gin.H{
				"title":    "Edit Character",
				"akun":     akun,
				"karakter": karakter,
			})
		})

		redirectLoginAutentikasi.POST("/karakter/", func(ctx *gin.Context) {
			db := database.GetDatabase()
			session := sessions.Default(ctx)
			user := session.Get("user")

			var akun utils.Akun
			db.First(&akun, user)

			status := ctx.PostForm("status")

			if status != "edit" && status != "buat" {
				ctx.Redirect(http.StatusFound, "/")
				return
			}

			karakter := utils.Karakter{}
			if status == "edit" {
				db.Where("id = ?", ctx.PostForm("idkarakter")).First(&karakter)

				if karakter.AkunID != akun.ID {
					ctx.Redirect(http.StatusFound, fmt.Sprintf("/editkarakter/%d", karakter.ID))
					return
				}
			}

			newKarakter := utils.Karakter{
				Nama:         ctx.PostForm("nama"),
				NamaLain:     ctx.PostForm("namalain"),
				Personalitas: ctx.PostForm("personalitas"),
				Kategori:     ctx.PostForm("kategori"),
				Chat:         ctx.PostForm("chat"),
				AkunID:       akun.ID,
				Akun:         akun,
			}

			if status == "edit" {
				newKarakter.ID = karakter.ID
				newKarakter.Gambar = karakter.Gambar
			}

			file, err := ctx.FormFile("foto")

			if err == nil {
				PathFile := "assets/gambar/" + strconv.Itoa(int(akun.ID)) + ".png"
				// PathFile := "assets/gambar/" + akun.ID.String() + ".png"
				ctx.SaveUploadedFile(file, PathFile)

				hasil := utils.UploadGambar(strconv.Itoa(int(akun.ID)))
				// hasil := utils.UploadGambar(akun.ID.String())
				newKarakter.Gambar = hasil.URL

				defer os.Remove(PathFile)
			}

			if status == "buat" {
				db.Create(&newKarakter)
			} else {
				db.Save(&newKarakter)
			}

			ctx.Redirect(http.StatusFound, fmt.Sprintf("/editkarakter/%d", newKarakter.ID))
		})

		HistoryChat := []*genai.Content{}
		redirectLoginAutentikasi.GET("/chat/:idchat", func(c *gin.Context) {
			db := database.GetDatabase()
			session := sessions.Default(c)

			akun := utils.DapatinAkun(db, session, &[]string{"Personalitas"})

			var karakter utils.Karakter
			db.Where("ID = ?", c.Param("idchat")).First(&karakter)

			var personalitas []utils.Personalitas
			db.Find(&personalitas, utils.Personalitas{AkunID: akun.ID})

			if karakter.ID == 0 {
				c.HTML(http.StatusNotFound, "error.html", gin.H{
					"title": "Error",
					"akun":  akun,
					"kode":  404,
					"isi":   "Character not found",
				})

				return
			}

			cs := utils.BuatChat(client, karakter, akun.Personalitas.DefaultPersonalitas(akun.Username), HistoryChat)

			if len(HistoryChat) <= 0 {
				HistoryChat = utils.DapatinSemuaPesan(cs)
			}

			c.HTML(http.StatusOK, "chat.html", gin.H{
				"title":        fmt.Sprintf("Chat with %s", karakter.Nama),
				"isi":          HistoryChat,
				"PanjangChat":  len(HistoryChat) - 1,
				"karakter":     karakter,
				"personalitas": personalitas,
				"akun":         akun,
			})
		})

		redirectLoginAutentikasi.POST("/chat/:idchat", func(c *gin.Context) {
			db := database.GetDatabase()

			session := sessions.Default(c)
			akun := utils.DapatinAkun(db, session, &[]string{"Personalitas"})

			var karakter utils.Karakter
			db.Where("ID = ?", c.Param("idchat")).First(&karakter)

			if karakter.ID == 0 {
				c.IndentedJSON(http.StatusNotFound, gin.H{
					"chat": nil,
				})

				return
			}

			cs := utils.BuatChat(client, karakter, akun.Personalitas.DefaultPersonalitas(akun.Username), HistoryChat)

			var newChat CHAT

			if err := c.BindJSON(&newChat); err != nil {
				c.IndentedJSON(http.StatusBadRequest, gin.H{
					"error": "Parameter missing",
				})
				return
			}

			resp, _ := utils.KirimPesan(cs, newChat.Chat)
			if resp == nil {
				c.IndentedJSON(http.StatusCreated, gin.H{
					"chat": nil,
				})
				return
			}

			// HistoryChat = utils.DapatinSemuaPesan(cs)
			HistoryChat = append(HistoryChat, []*genai.Content{
				{
					Parts: []genai.Part{
						genai.Text(newChat.Chat),
					},
					Role: "user",
				},
				{
					Parts: []genai.Part{
						resp.Candidates[0].Content.Parts[0],
					},
					Role: "model",
				},
			}...)

			c.IndentedJSON(http.StatusCreated, gin.H{
				"chat": resp.Candidates[0].Content.Parts[0],
			})
		})

		redirectLoginAutentikasi.POST("/chat/:idchat/ulangipesan", func(ctx *gin.Context) {
			db := database.GetDatabase()

			session := sessions.Default(ctx)
			akun := utils.DapatinAkun(db, session, &[]string{"Personalitas"})

			var karakter utils.Karakter
			db.Where("ID = ?", ctx.Param("idchat")).First(&karakter)

			if karakter.ID == 0 {
				ctx.IndentedJSON(http.StatusNotFound, gin.H{
					"chat": nil,
				})

				return
			}

			cs := utils.BuatChat(client, karakter, akun.Personalitas.DefaultPersonalitas(akun.Username), HistoryChat)

			resp, _ := utils.UlangiJawaban(cs)
			if resp == nil {
				ctx.IndentedJSON(http.StatusCreated, gin.H{
					"chat": nil,
				})
				return
			}

			// HistoryChat = append(HistoryChat, &genai.Content{
			// 	Parts: []genai.Part{
			// 		resp.Candidates[0].Content.Parts[0],
			// 	},
			// 	Role: "model",
			// })

			ctx.IndentedJSON(http.StatusCreated, gin.H{
				"chat": resp.Candidates[0].Content.Parts[0],
			})
		})

		redirectLoginAutentikasi.POST("/chat/:idchat/sarankalimat", func(ctx *gin.Context) {
			db := database.GetDatabase()

			session := sessions.Default(ctx)
			akun := utils.DapatinAkun(db, session, &[]string{"Personalitas"})

			var karakter utils.Karakter
			db.Where("ID = ?", ctx.Param("idchat")).First(&karakter)

			if karakter.ID == 0 {
				ctx.IndentedJSON(http.StatusNotFound, gin.H{
					"chat": nil,
				})

				return
			}

			cs := utils.BuatChat(client, karakter, akun.Personalitas.DefaultPersonalitas(akun.Username), HistoryChat)

			resp := utils.SaranKalimat(client, cs)
			if resp == nil {
				ctx.IndentedJSON(http.StatusCreated, gin.H{
					"chat": nil,
				})

				return
			}

			ctx.IndentedJSON(http.StatusCreated, gin.H{
				"chat": resp.Candidates[0].Content.Parts[0],
			})
		})

		redirectLoginAutentikasi.POST("/personalitas", func(ctx *gin.Context) {
			var newPersonalitas StatusPeronsalitas

			if err := ctx.ShouldBindJSON(&newPersonalitas); err != nil {
				fmt.Println(err)
				ctx.IndentedJSON(http.StatusCreated, gin.H{
					"status": "Error",
				})
				return
			}

			if newPersonalitas.Status != "buat" && newPersonalitas.Status != "edit" && newPersonalitas.Status != "pilih" {
				ctx.IndentedJSON(http.StatusCreated, gin.H{
					"status": "Error",
				})
				return
			}

			db := database.GetDatabase()
			session := sessions.Default(ctx)

			akun := utils.DapatinAkun(db, session, &[]string{"Personalitas"})

			if newPersonalitas.Status == "pilih" {
				checkPersonalitas := utils.Personalitas{}
				db.Where("id = ?", newPersonalitas.ID).First(&checkPersonalitas)

				if checkPersonalitas.ID == 0 {
					ctx.IndentedJSON(http.StatusCreated, gin.H{
						"status": "Error",
					})
					return
				}

				// fmt.Println("PERSONALITAS BARU: " + newPersonalitas.ID)
				// fmt.Println("CHECKPERSONALITAS: " + strconv.Itoa(int(checkPersonalitas.ID)))
				// fmt.Println("SEBELUM AKUN PERSONALITAS: " + strconv.Itoa(int(akun.PersonalitasID)))

				// akun.Personalitas.ID = checkPersonalitas.ID
				// db.Session(&gorm.Session{FullSaveAssociations: true}).Update(&akun)
				// db.Model(&akun).Association("Personalitas").Clear()
				db.Model(&akun).Update("Personalitas", utils.Personalitas{ID: checkPersonalitas.ID})
				// db.Model(&akun).Update("PersonalitasID", checkPersonalitas.ID)
				// db.Save(&akun)

				// fmt.Println("AKUN PERSONALITAS: " + strconv.Itoa(int(akun.PersonalitasID)))

				ctx.IndentedJSON(http.StatusCreated, gin.H{
					"status": "Success",
				})

				return
			}

			if newPersonalitas.Nama == "" || newPersonalitas.Personalitas == "" {
				ctx.IndentedJSON(http.StatusCreated, gin.H{
					"status": "Error",
				})
				return
			}

			personalitas := utils.Personalitas{
				Nama:         newPersonalitas.Nama,
				Personalitas: newPersonalitas.Personalitas,
				AkunID:       akun.ID,
			}

			if newPersonalitas.Status == "buat" {
				db.Create(&personalitas)
			} else if newPersonalitas.Status == "edit" {
				db.Save(&personalitas)
			}

			ctx.IndentedJSON(http.StatusCreated, gin.H{
				"status":       "Berhasil",
				"nama":         personalitas.Nama,
				"personalitas": personalitas.Personalitas,
				"id":           personalitas.ID,
			})
		})
	}

	redirectHomeAutentikasi := r.Group("/")
	redirectHomeAutentikasi.Use(CheckAutentikasi("login"))
	{
		redirectHomeAutentikasi.GET("/login", func(ctx *gin.Context) {
			session := sessions.Default(ctx)

			ctx.HTML(http.StatusOK, "login.html", gin.H{
				"title": "Login",
				"flash": session.Flashes("Error"),
			})
		})

		redirectHomeAutentikasi.GET("/register", func(ctx *gin.Context) {
			session := sessions.Default(ctx)

			ctx.HTML(http.StatusOK, "register.html", gin.H{
				"title": "Register",
				"flash": session.Flashes("Error"),
			})
		})

		redirectHomeAutentikasi.POST("/login", func(ctx *gin.Context) {
			session := sessions.Default(ctx)
			nama_akun := ctx.PostForm("nama_akun")
			password := ctx.PostForm("password")

			if strings.Trim(nama_akun, " ") == "" || strings.Trim(password, " ") == "" {
				ctx.Redirect(http.StatusMovedPermanently, "/login")
				return
			}

			db := database.GetDatabase()
			akun := utils.Akun{}

			db.First(&akun, "email = ? OR username = ?", nama_akun, nama_akun)

			fmt.Println("AKUN LOGIN, USERNAME: " + akun.Username)
			fmt.Println(akun.ID)

			// akun.ID == 0 ||
			if akun.ID == 0 || !utils.CheckPasswordHash(password, akun.Password) {
				session.AddFlash("Email or password is incorrect", "Error")
				session.Save()

				ctx.Redirect(http.StatusMovedPermanently, "/login")
				return
			}

			// if email != "tester144@gmail.com" || password != "123456" {
			// 	ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
			// 	return
			// }

			// Save the username in the session
			session.Set("user", akun.ID)
			session.Save()

			ctx.Redirect(http.StatusFound, "/")
		})

		redirectHomeAutentikasi.POST("/register", func(ctx *gin.Context) {
			session := sessions.Default(ctx)
			username := ctx.PostForm("username")
			email := ctx.PostForm("email")
			password := ctx.PostForm("password")

			if strings.Trim(email, " ") == "" || strings.Trim(password, " ") == "" || strings.Trim(username, " ") == "" {
				ctx.Redirect(http.StatusMovedPermanently, "/register")
				return
			}

			db := database.GetDatabase()

			akunDiMasuin := utils.Akun{}
			db.First(&akunDiMasuin, "email = ? OR username = ?", email, username)

			fmt.Println("REGISTER AKUN: " + akunDiMasuin.Username)

			if akunDiMasuin.Email != "" || akunDiMasuin.Username != "" {
				if akunDiMasuin.Email != "" {
					session.AddFlash("Email have already used!", "Error")
				} else {
					session.AddFlash("Username have already used!", "Error")
				}
				session.Save()

				ctx.Redirect(http.StatusMovedPermanently, "/register")
				return
			}

			hashPassword, err := utils.HashPassword(password)
			if err != nil {
				ctx.Redirect(http.StatusMovedPermanently, "/register")
				return
			}

			registerAkun := utils.Akun{Username: username, Email: email, Password: hashPassword, ImageURL: "/assets/no-users.png"}

			db.Create(&registerAkun)
			fmt.Println("AAAAAH")

			session.Set("user", registerAkun.ID)
			if err := session.Save(); err != nil {
				fmt.Println(err)
				ctx.Redirect(http.StatusFound, "/register")
				return
			}

			ctx.Redirect(http.StatusFound, "/")
		})
	}

	r.Run()
}
