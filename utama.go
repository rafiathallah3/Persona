package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"strings"

	"persona/database"
	"persona/routes"
	"persona/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	// "github.com/google/uuid"
)

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

func main() {
	// sk-proj-pSehnOF8HPuzFkEU2NK0T3BlbkFJxrUc1Pu0PXkHruHhzjbN
	// gmCePyjssc9j9I5hw29ymg

	// postgresql://rapithon:gmCePyjssc9j9I5hw29ymg@per-chat-7248.6xw.aws-ap-southeast-1.cockroachlabs.cloud:26257/defaultdb?sslmode=verify-full

	database.Connect()
	utils.InitCloudinary()

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
	// r.MaxMultipartMemory = 20 << 20

	r.GET("/", func(ctx *gin.Context) {
		db := database.GetDatabase()
		session := sessions.Default(ctx)

		akun := utils.DapatinAkun(db, session, nil)

		var SemuaKarakter []utils.Karakter
		db.Find(&SemuaKarakter)

		var rawSemuaChat []utils.KarakterChat
		db.Preload("History").Preload("Karakter").Find(&rawSemuaChat, utils.KarakterChat{PechatID: akun.ID})

		var listChat []utils.ListChat
		for _, value := range rawSemuaChat {
			listChat = append(listChat, utils.ListChat{
				IDChat:       value.ID,
				IDKarakter:   value.Karakter.ID,
				ChatTerakhir: value.History[len(value.History)-1].Chat,
				Nama:         value.Karakter.Nama,
				Gambar:       value.Karakter.Gambar,
				Tag:          value.Karakter.Kategori,
				CreatedAt:    value.Karakter.CreatedAt,
			})
		}

		for i := 0; i < len(listChat)-1; i++ {
			for j := i + 1; j < len(listChat); j++ {
				if listChat[i].CreatedAt.After(listChat[j].CreatedAt) {
					listChat[i], listChat[j] = listChat[j], listChat[i]
				}
			}
		}

		ctx.HTML(http.StatusOK, "index.html", gin.H{
			"title":     "Main",
			"akun":      akun,
			"karakter":  SemuaKarakter,
			"semuaChat": listChat,
		})
	})

	r.GET("/search", func(ctx *gin.Context) {
		db := database.GetDatabase()
		session := sessions.Default(ctx)

		akun := utils.DapatinAkun(db, session, nil)

		var SemuaKarakter []utils.Karakter
		db.Find(&SemuaKarakter)

		ctx.HTML(http.StatusOK, "cari.html", gin.H{
			"title":    "Search Character",
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
	redirectLoginAutentikasi.Use(routes.DapatinAkun())
	redirectLoginAutentikasi.Use(routes.CheckAutentikasi("akses"))
	{
		redirectLoginAutentikasi.GET("/buatkarakter/", func(ctx *gin.Context) {
			akunRaw, _ := ctx.Get("akun")
			akun := akunRaw.(utils.Akun)

			session := sessions.Default(ctx)

			ctx.HTML(http.StatusOK, "karakter.html", gin.H{
				"title": "Create Character",
				"akun":  akun,
				"flash": session.Flashes(),
			})
		})

		redirectLoginAutentikasi.GET("/editkarakter/:idkarakter", func(ctx *gin.Context) {
			dbRaw, _ := ctx.Get("db")
			db := dbRaw.(*gorm.DB)

			akunRaw, _ := ctx.Get("akun")
			akun := akunRaw.(utils.Akun)

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
			db, akun := routes.InitAkunDB(ctx)

			session := sessions.Default(ctx)

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

				statusUpdate := ctx.PostForm("status_edit")

				if statusUpdate == "Delete" {
					utils.HapusGambar(strconv.FormatUint(karakter.ID, 10))
					db.Unscoped().Delete(&karakter)

					session.AddFlash("Character successfully been deleted", "Sukses")
					session.Save()
					ctx.Redirect(http.StatusFound, "/buatkarakter")
					return
				}
			}

			kategori := ctx.PostForm("kategori")
			if !slices.Contains(utils.Kategori, kategori) {
				ctx.Redirect(http.StatusFound, "/buatkarakter")
				return
			}

			newKarakter := utils.Karakter{
				Nama:         ctx.PostForm("nama"),
				NamaLain:     ctx.PostForm("namalain"),
				Personalitas: ctx.PostForm("personalitas"),
				Kategori:     kategori,
				Deskripsi:    ctx.PostForm("deskripsi"),
				Chat:         ctx.PostForm("chat"),
				AkunID:       akun.ID,
				Akun:         akun,
			}

			if status == "edit" {
				newKarakter.ID = karakter.ID
				newKarakter.Gambar = karakter.Gambar
			}

			file, err := ctx.FormFile("foto")

			if err != nil && status == "buat" {
				newKarakter.Gambar = "/assets/no-users.png"
			}

			if status == "buat" {
				db.Create(&newKarakter)
			} else {
				db.Save(&newKarakter)
			}

			if err == nil {
				PathFile := "assets/gambar/" + strconv.Itoa(int(newKarakter.ID)) + ".png"
				// PathFile := "assets/gambar/" + newKarakter.ID.String() + ".png"
				ctx.SaveUploadedFile(file, PathFile)

				hasil := utils.UploadGambar(strconv.Itoa(int(newKarakter.ID)))
				// hasil := utils.UploadGambar(newKarakter.ID.String())
				newKarakter.Gambar = hasil.URL

				db.Model(&newKarakter).Update("Gambar", hasil.URL)

				defer os.Remove(PathFile)
			}

			ctx.Redirect(http.StatusFound, fmt.Sprintf("/editkarakter/%d", newKarakter.ID))
		})

		// HistoryChat := []*genai.Content{}
		redirectLoginAutentikasi.GET("/chat/:idkarakter/*idchat", func(c *gin.Context) {
			db, akun := routes.InitAkunDB(c)
			dataChat, err := routes.InitChat(c)

			if err != nil {
				c.HTML(http.StatusNotFound, "error.html", gin.H{
					"title": "Error",
					"akun":  akun,
					"kode":  404,
					"isi":   "Character not found",
				})

				return
			}

			if dataChat.KarakterChat.ID == 0 && c.Param("idchat") != "/" {
				c.HTML(http.StatusNotFound, "error.html", gin.H{
					"title": "Error",
					"akun":  akun,
					"kode":  404,
					"isi":   "Chat not found",
				})

				return
			}

			var personalitas []utils.Personalitas
			db.Find(&personalitas, utils.Personalitas{AkunID: akun.ID})

			// var semua_isiChat []utils.IsiChat
			// db.Where("room_chat_id = ?", karakterChat.ID).Find(&semua_isiChat)

			var rawSemuaChat []utils.KarakterChat
			db.Preload("History").Find(&rawSemuaChat, utils.KarakterChat{KarakterID: dataChat.Karakter.ID, PechatID: akun.ID})

			var listChat []utils.ListChat
			for _, value := range rawSemuaChat {
				listChat = append(listChat, utils.ListChat{
					IDChat:       value.ID,
					ChatTerakhir: value.History[len(value.History)-1].Chat,
				})
			}

			genAIHistoryChat, dataHistoryChat := utils.DapatinHistoryKarakter(dataChat.KarakterChat)

			cs := utils.BuatChat(client, dataChat.Karakter, akun.Personalitas.DefaultPersonalitas(akun.Username), genAIHistoryChat)

			if len(genAIHistoryChat) <= 0 {
				for _, value := range utils.DapatinSemuaPesan(cs) {
					dataHistoryChat = append(dataHistoryChat, utils.DataHistoryChat{
						ID:    0,
						Chat:  fmt.Sprintf("%v", value.Parts[0]),
						Role:  value.Role,
						Waktu: time.Now(),
					})
				}
			}

			c.HTML(http.StatusOK, "chat.html", gin.H{
				"title":        fmt.Sprintf("Chat with %s", dataChat.Karakter.Nama),
				"isi":          dataHistoryChat,
				"PanjangChat":  len(dataHistoryChat) - 1,
				"karakter":     dataChat.Karakter,
				"personalitas": personalitas,
				"semuaChat":    listChat,
				"chatid":       strings.ReplaceAll(c.Param("idchat"), "/", ""),
				"akun":         akun,
			})
		})

		redirectLoginAutentikasi.POST("/api/chat/", func(c *gin.Context) {
			// var newChat utils.PostChat

			// if err := c.BindJSON(&newChat); err != nil {
			// 	c.IndentedJSON(http.StatusBadRequest, gin.H{
			// 		"error": "Parameter missing",
			// 	})
			// 	return
			// }

			db, akun := routes.InitAkunDB(c)
			dataChat, err := routes.InitChat(c)

			if err != nil {
				c.IndentedJSON(http.StatusNotFound, gin.H{
					"chat": nil,
				})

				return
			}

			indexIsiChat := 0

			if dataChat.KarakterChat.ID == 0 {
				dataChat.KarakterChat.KarakterID = dataChat.Karakter.ID
				dataChat.KarakterChat.PechatID = akun.ID
				// dataChat.KarakterChat.History = []utils.IsiChat{newIsiChat}
				db.Create(&dataChat.KarakterChat)

				newIsiChat := utils.IsiChat{Chat: dataChat.Karakter.RenderChat(akun.Personalitas.DefaultPersonalitas(akun.Username).Nama), Role: "model", RoomChatID: dataChat.KarakterChat.ID, DariPecatID: akun.ID, Posisi: 1}
				db.Create(&newIsiChat)

				indexIsiChat++
			}

			genAIHistoryChat, _ := utils.DapatinHistoryKarakter(dataChat.KarakterChat)
			indexIsiChat += len(genAIHistoryChat)

			cs := utils.BuatChat(client, dataChat.Karakter, akun.Personalitas.DefaultPersonalitas(akun.Username), genAIHistoryChat)

			resp, _ := utils.KirimPesan(cs, dataChat.PostChat.Chat)
			if resp == nil {
				c.IndentedJSON(http.StatusCreated, gin.H{
					"chat": nil,
				})
				return
			}

			db.Model(&dataChat.KarakterChat).Association("History").Append([]utils.IsiChat{
				{Chat: dataChat.PostChat.Chat, Role: "user", RoomChatID: dataChat.KarakterChat.ID, DariPecatID: akun.ID, Posisi: uint8(indexIsiChat + 1)},
				{Chat: fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), Role: "model", RoomChatID: dataChat.KarakterChat.ID, DariPecatID: akun.ID, Posisi: uint8(indexIsiChat + 2)},
			})

			isiChatID := strconv.FormatUint(dataChat.KarakterChat.History[len(dataChat.KarakterChat.History)-1].ID, 10)
			c.IndentedJSON(http.StatusCreated, gin.H{
				"chat": resp.Candidates[0].Content.Parts[0],
				"id":   isiChatID,
			})
		})

		redirectLoginAutentikasi.POST("/api/chat/buatchat", func(ctx *gin.Context) {
			db, akun := routes.InitAkunDB(ctx)
			dataChat, err := routes.InitChat(ctx)

			karakterID := strconv.FormatUint(dataChat.Karakter.ID, 10)
			if err != nil || dataChat.KarakterChat.ID == 0 {
				ctx.Redirect(http.StatusMovedPermanently, "/chat/"+karakterID)
				return
			}

			status := ctx.PostForm("status")
			karakterChat := utils.KarakterChat{
				KarakterID: dataChat.Karakter.ID,
				PechatID:   akun.ID,
			}

			db.Create(&karakterChat)

			if status == "clone" && len(dataChat.KarakterChat.History) > 1 {
				for i, v := range dataChat.KarakterChat.History {
					newIsiChat := utils.IsiChat{Chat: v.Chat, Role: v.Role, RoomChatID: karakterChat.ID, DariPecatID: akun.ID, Posisi: uint8(i + 1)}
					db.Create(&newIsiChat)
				}
			}

			if status == "baru" {
				newIsiChat := utils.IsiChat{Chat: dataChat.Karakter.RenderChat(akun.Personalitas.DefaultPersonalitas(akun.Username).Nama), Role: "model", RoomChatID: karakterChat.ID, DariPecatID: akun.ID, Posisi: 1}
				db.Create(&newIsiChat)
			}

			ctx.Redirect(http.StatusMovedPermanently, "/chat/"+karakterID+"/"+strconv.FormatUint(karakterChat.ID, 10))
		})

		redirectLoginAutentikasi.POST("/api/chat/ulangipesan", func(ctx *gin.Context) {
			db, akun := routes.InitAkunDB(ctx)
			dataChat, err := routes.InitChat(ctx)

			if err != nil {
				ctx.IndentedJSON(http.StatusNotFound, gin.H{
					"chat": nil,
				})

				return
			}

			genAIHistoryChat, _ := utils.DapatinHistoryKarakter(dataChat.KarakterChat)

			cs := utils.BuatChat(client, dataChat.Karakter, akun.Personalitas.DefaultPersonalitas(akun.Username), genAIHistoryChat)

			resp, _ := utils.UlangiJawaban(cs)
			if resp == nil {
				ctx.IndentedJSON(http.StatusCreated, gin.H{
					"chat": nil,
				})
				return
			}

			isiCharDiPilih := dataChat.KarakterChat.History[len(dataChat.KarakterChat.History)-1]
			ChatDulu := isiCharDiPilih.Chat
			db.Model(&isiCharDiPilih).Update("chat", fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]))

			// HistoryChat = append(HistoryChat, &genai.Content{
			// 	Parts: []genai.Part{
			// 		resp.Candidates[0].Content.Parts[0],
			// 	},
			// 	Role: "model",
			// })

			ctx.IndentedJSON(http.StatusCreated, gin.H{
				"chat":     resp.Candidates[0].Content.Parts[0],
				"chatDulu": ChatDulu,
				"id":       strconv.FormatUint(isiCharDiPilih.ID, 10),
			})
		})

		redirectLoginAutentikasi.POST("/api/chat/sarankalimat", func(ctx *gin.Context) {
			_, akun := routes.InitAkunDB(ctx)
			dataChat, err := routes.InitChat(ctx)

			if err != nil {
				ctx.IndentedJSON(http.StatusNotFound, gin.H{
					"chat": nil,
				})

				return
			}

			genAIHistoryChat, _ := utils.DapatinHistoryKarakter(dataChat.KarakterChat)

			cs := utils.BuatChat(client, dataChat.Karakter, akun.Personalitas.DefaultPersonalitas(akun.Username), genAIHistoryChat)

			resp := utils.SaranKalimat(client, akun.Personalitas.DefaultPersonalitas(akun.Username), cs)
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

		redirectLoginAutentikasi.POST("/api/chat/hapuspesan", func(ctx *gin.Context) {
			db, _ := routes.InitAkunDB(ctx)
			dataChat, err := routes.InitChat(ctx)

			if err != nil || len(dataChat.KarakterChat.History) <= 0 {
				ctx.IndentedJSON(http.StatusNotFound, gin.H{
					"status": "error",
				})

				return
			}

			db.Unscoped().Delete(&dataChat.KarakterChat.History[len(dataChat.KarakterChat.History)-1])
			db.Unscoped().Delete(&dataChat.KarakterChat.History[len(dataChat.KarakterChat.History)-2])

			ctx.IndentedJSON(http.StatusCreated, gin.H{
				"status": "sukses",
			})
		})

		redirectLoginAutentikasi.POST("/api/chat/hapuschat", func(ctx *gin.Context) {
			db, _ := routes.InitAkunDB(ctx)
			dataChat, err := routes.InitChat(ctx)

			karakterID := strconv.FormatUint(dataChat.Karakter.ID, 10)
			if err != nil || dataChat.KarakterChat.ID == 0 {
				ctx.Redirect(http.StatusMovedPermanently, "/chat/"+karakterID)
				return
			}

			db.Unscoped().Delete(&dataChat.KarakterChat)

			ctx.Redirect(http.StatusMovedPermanently, "/chat/"+karakterID)
		})

		redirectLoginAutentikasi.POST("/api/chat/updatepesan", func(ctx *gin.Context) {
			db, akun := routes.InitAkunDB(ctx)
			dataChat, err := routes.InitChat(ctx)

			if err != nil || len(dataChat.KarakterChat.History) <= 0 {
				ctx.IndentedJSON(http.StatusNotFound, gin.H{
					"status": "error",
				})

				return
			}

			isiChat := utils.IsiChat{}
			db.Where("id = ? AND dari_pecat_id = ?", dataChat.PostChat.PesanID, akun.ID).First(&isiChat)

			if isiChat.ID == 0 {
				ctx.IndentedJSON(http.StatusNotFound, gin.H{
					"status": "error",
				})

				return
			}

			db.Model(&isiChat).Update("chat", dataChat.PostChat.Chat)

			ctx.IndentedJSON(http.StatusCreated, gin.H{
				"status": "sukses",
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

			dbRaw, _ := ctx.Get("db")
			db := dbRaw.(*gorm.DB)

			akunRaw, _ := ctx.Get("akun")
			akun := akunRaw.(utils.Akun)

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

			fmt.Println("STATUS: " + newPersonalitas.Status)
			fmt.Println("ID!!!: " + newPersonalitas.ID)

			if newPersonalitas.Status == "buat" {
				db.Create(&personalitas)
			} else if newPersonalitas.Status == "edit" {
				checkPersonalitas := utils.Personalitas{}
				db.Where("id = ?", newPersonalitas.ID).First(&checkPersonalitas)

				if checkPersonalitas.ID == 0 {
					ctx.IndentedJSON(http.StatusCreated, gin.H{
						"status": "Error",
					})
					return
				}

				personalitas.ID = checkPersonalitas.ID

				db.Save(&personalitas)
			}

			ctx.IndentedJSON(http.StatusCreated, gin.H{
				"status":       "Berhasil",
				"nama":         personalitas.Nama,
				"personalitas": personalitas.Personalitas,
				"id":           strconv.FormatUint(personalitas.ID, 10),
				// "id":           strconv.Itoa(int(personalitas.ID)),
			})
		})
	}

	redirectHomeAutentikasi := r.Group("/")
	redirectHomeAutentikasi.Use(routes.CheckAutentikasi("login"))
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
