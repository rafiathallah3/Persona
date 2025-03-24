package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"strings"

	"persona/api/auth"
	"persona/api/chat"
	"persona/database"
	"persona/database/models"
	"persona/routes"
	"persona/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	// "github.com/google/uuid"
)

type StatusPeronsalitas struct {
	Status       string `json:"status"`
	ID           string `json:"id"`
	Nama         string `json:"nama"`
	Personalitas string `json:"personalitas"`
}

var db *gorm.DB
var client *genai.Client
var rds *redis.Client
var redis_ctx = context.Background()

/* ##########     PAGE    ########### */
func IndexPage(ctx *gin.Context) {
	session := sessions.Default(ctx)

	akun := auth.DapatinAkun(db, session, nil)

	var SemuaKarakter []models.Karakter
	db.Find(&SemuaKarakter)

	var rawSemuaChat []models.KarakterChat
	db.Preload("History").Preload("Karakter").Where("pechat_id = ?", akun.ID).Find(&rawSemuaChat)

	var listChat []models.ListChat
	for _, value := range rawSemuaChat {
		listChat = append(listChat, models.ListChat{
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
}

func SearchPage(ctx *gin.Context) {
	session := sessions.Default(ctx)

	akun := auth.DapatinAkun(db, session, nil)

	var SemuaKarakter []models.Karakter

	CariNama := ctx.Query("nama")
	CariGenre := ctx.Query("genre")

	cariKarakter := models.Karakter{}
	query := db

	if CariNama != "" {
		cariKarakter.Nama = CariNama
		query = query.Where("nama LIKE ?", "%"+CariNama+"%")
	}

	if CariGenre != "" {
		cariKarakter.Kategori = CariGenre
		query = query.Where("kategori = ?", CariGenre)
	}

	query.Find(&SemuaKarakter)

	ctx.HTML(http.StatusOK, "cari.html", gin.H{
		"title":        "Search Character",
		"akun":         akun,
		"karakter":     SemuaKarakter,
		"karakterCari": cariKarakter,
	})
}

func ProfilePage(c *gin.Context) {
	c.HTML(http.StatusOK, "profile.html", gin.H{
		"title": "Profile",
	})
}

func BuatKarakterPage(ctx *gin.Context) {
	akunRaw, _ := ctx.Get("akun")
	akun := akunRaw.(models.Akun)

	session := sessions.Default(ctx)
	flash := session.Get("flash")
	session.Delete("flash")
	session.Save()

	ctx.HTML(http.StatusOK, "karakter.html", gin.H{
		"title": "Create Character",
		"akun":  akun,
		"flash": flash,
	})
}

func EditKarakterPage(ctx *gin.Context) {
	dbRaw, _ := ctx.Get("db")
	db := dbRaw.(*gorm.DB)

	akunRaw, _ := ctx.Get("akun")
	akun := akunRaw.(models.Akun)

	var karakter models.Karakter
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
}

func KarakterChatPage(c *gin.Context) {
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

	link_idChat := c.Param("idchat")
	if dataChat.KarakterChat.ID == 0 && link_idChat != "/" {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"title": "Error",
			"akun":  akun,
			"kode":  404,
			"isi":   "Chat not found",
		})

		return
	}

	var personalitas []models.Personalitas
	db.Find(&personalitas, models.Personalitas{AkunID: akun.ID})

	// var semua_isiChat []models.IsiChat
	// db.Where("room_chat_id = ?", karakterChat.ID).Find(&semua_isiChat)

	var rawSemuaChat []models.KarakterChat
	db.Preload("History").Find(&rawSemuaChat, models.KarakterChat{KarakterID: dataChat.Karakter.ID, PechatID: akun.ID})

	var listChat []models.ListChat
	for _, value := range rawSemuaChat {
		listChat = append(listChat, models.ListChat{
			IDChat:       value.ID,
			ChatTerakhir: value.History[len(value.History)-1].Chat,
		})
	}

	genAIHistoryChat, dataHistoryChat := chat.DapatinHistoryKarakter(dataChat.KarakterChat)

	cs := chat.BuatChat(client, dataChat.Karakter, akun.Personalitas.DefaultPersonalitas(akun.Username), genAIHistoryChat)

	if len(genAIHistoryChat) <= 0 {
		for _, value := range chat.DapatinSemuaPesan(cs) {
			dataHistoryChat = append(dataHistoryChat, models.DataHistoryChat{
				ID:    0,
				Chat:  fmt.Sprintf("%v", value.Parts[0]),
				Role:  value.Role,
				Waktu: time.Now(),
			})
		}
	}

	val, err := rds.LRange(redis_ctx, fmt.Sprintf("%d-%d", akun.ID, dataChat.Karakter.ID), 0, -1).Result()
	if err == nil {
		fmt.Println("DAPATIN UPDATED CHAT", val)
	}

	c.HTML(http.StatusOK, "chat.html", gin.H{
		"title":        fmt.Sprintf("Chat with %s", dataChat.Karakter.Nama),
		"isi":          dataHistoryChat,
		"PanjangChat":  len(dataHistoryChat) - 1,
		"karakter":     dataChat.Karakter,
		"personalitas": personalitas,
		"semuaChat":    listChat,
		"ulangiPesan":  val,
		"chatid":       dataChat.KarakterChat.ID,
		"akun":         akun,
	})
}

func LoginPage(ctx *gin.Context) {
	session := sessions.Default(ctx)

	flash := session.Get("flash")
	session.Delete("flash")
	session.Save()

	ctx.HTML(http.StatusOK, "login.html", gin.H{
		"title": "Login",
		"flash": flash,
	})
}

func RegisterPage(ctx *gin.Context) {
	session := sessions.Default(ctx)
	flash := session.Get("flash")
	session.Delete("flash")
	session.Save()

	ctx.HTML(http.StatusOK, "register.html", gin.H{
		"title": "Register",
		"flash": flash,
	})
}

/* ##########     HANDLER    ########### */

func KarakterHandler(ctx *gin.Context) {
	db, akun := routes.InitAkunDB(ctx)

	session := sessions.Default(ctx)

	status := ctx.PostForm("status")

	if status != "edit" && status != "buat" {
		ctx.Redirect(http.StatusFound, "/")
		return
	}

	karakter := models.Karakter{}
	if status == "edit" {
		db.Where("id = ?", ctx.PostForm("idkarakter")).First(&karakter)

		if karakter.AkunID != akun.ID {
			ctx.Redirect(http.StatusFound, fmt.Sprintf("/editkarakter/%d", karakter.ID))
			return
		}

		statusUpdate := ctx.PostForm("status_edit")

		if statusUpdate == "Delete" {
			database.HapusGambar(strconv.FormatUint(karakter.ID, 10))
			db.Unscoped().Delete(&karakter)

			session.Set("flash", "Character has been successfully deleted!")
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

	newKarakter := models.Karakter{
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

		hasil := database.UploadGambar(strconv.Itoa(int(newKarakter.ID)))
		// hasil := utils.UploadGambar(newKarakter.ID.String())
		newKarakter.Gambar = hasil.URL

		db.Model(&newKarakter).Update("Gambar", hasil.URL)

		defer os.Remove(PathFile)
	}

	ctx.Redirect(http.StatusFound, fmt.Sprintf("/editkarakter/%d", newKarakter.ID))
}

func BuatChatHandler(ctx *gin.Context) {
	db, akun := routes.InitAkunDB(ctx)
	dataChat, err := routes.InitChat(ctx)

	karakterID := strconv.FormatUint(dataChat.Karakter.ID, 10)
	if err != nil || dataChat.KarakterChat.ID == 0 {
		ctx.Redirect(http.StatusMovedPermanently, "/chat/"+karakterID)
		return
	}

	status := ctx.PostForm("status")
	karakterChat := models.KarakterChat{
		KarakterID: dataChat.Karakter.ID,
		PechatID:   akun.ID,
	}

	db.Create(&karakterChat)

	if status == "clone" && len(dataChat.KarakterChat.History) > 1 {
		for i, v := range dataChat.KarakterChat.History {
			newIsiChat := models.IsiChat{Chat: v.Chat, Role: v.Role, RoomChatID: karakterChat.ID, DariPecatID: akun.ID, Posisi: uint8(i + 1)}
			db.Create(&newIsiChat)
		}
	}

	if status == "baru" {
		newIsiChat := models.IsiChat{Chat: dataChat.Karakter.RenderChat(akun.Personalitas.DefaultPersonalitas(akun.Username).Nama), Role: "model", RoomChatID: karakterChat.ID, DariPecatID: akun.ID, Posisi: 1}
		db.Create(&newIsiChat)
	}

	ctx.Redirect(http.StatusMovedPermanently, "/chat/"+karakterID+"/"+strconv.FormatUint(karakterChat.ID, 10))
}

func ChatHandler(c *gin.Context) {
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

		newIsiChat := models.IsiChat{Chat: dataChat.Karakter.RenderChat(akun.Personalitas.DefaultPersonalitas(akun.Username).Nama), Role: "model", RoomChatID: dataChat.KarakterChat.ID, DariPecatID: akun.ID, Posisi: 1}
		db.Create(&newIsiChat)

		indexIsiChat++
	}

	rds.Del(redis_ctx, fmt.Sprintf("%d-%d", akun.ID, dataChat.Karakter.ID))

	genAIHistoryChat, _ := chat.DapatinHistoryKarakter(dataChat.KarakterChat)
	indexIsiChat += len(genAIHistoryChat)

	cs := chat.BuatChat(client, dataChat.Karakter, akun.Personalitas.DefaultPersonalitas(akun.Username), genAIHistoryChat)

	resp, _ := chat.KirimPesan(cs, dataChat.PostChat.Chat)
	if resp == nil {
		c.IndentedJSON(http.StatusCreated, gin.H{
			"chat": nil,
		})
		return
	}

	rds.LPush(redis_ctx, fmt.Sprintf("%d-%d", akun.ID, dataChat.Karakter.ID), fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]))

	db.Model(&dataChat.KarakterChat).Association("History").Append([]models.IsiChat{
		{Chat: dataChat.PostChat.Chat, Role: "user", RoomChatID: dataChat.KarakterChat.ID, DariPecatID: akun.ID, Posisi: uint8(indexIsiChat + 1)},
		{Chat: fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), Role: "model", RoomChatID: dataChat.KarakterChat.ID, DariPecatID: akun.ID, Posisi: uint8(indexIsiChat + 2)},
	})

	isiChatID := strconv.FormatUint(dataChat.KarakterChat.History[len(dataChat.KarakterChat.History)-1].ID, 10)
	c.IndentedJSON(http.StatusCreated, gin.H{
		"karakterChatID": dataChat.KarakterChat.ID,
		"chat":           resp.Candidates[0].Content.Parts[0],
		"id":             isiChatID,
	})
}

func HapusChatHandler(ctx *gin.Context) {
	db, _ := routes.InitAkunDB(ctx)
	dataChat, err := routes.InitChat(ctx)

	karakterID := strconv.FormatUint(dataChat.Karakter.ID, 10)
	if err != nil || dataChat.KarakterChat.ID == 0 {
		ctx.Redirect(http.StatusMovedPermanently, "/chat/"+karakterID)
		return
	}

	db.Unscoped().Delete(&dataChat.KarakterChat)

	ctx.Redirect(http.StatusMovedPermanently, "/chat/"+karakterID)
}

func UlangiPesanHandler(ctx *gin.Context) {
	db, akun := routes.InitAkunDB(ctx)
	dataChat, err := routes.InitChat(ctx)

	if err != nil {
		ctx.IndentedJSON(http.StatusNotFound, gin.H{
			"chat": nil,
		})

		return
	}

	genAIHistoryChat, _ := chat.DapatinHistoryKarakter(dataChat.KarakterChat)

	cs := chat.BuatChat(client, dataChat.Karakter, akun.Personalitas.DefaultPersonalitas(akun.Username), genAIHistoryChat)

	resp, _ := chat.UlangiJawaban(cs)
	if resp == nil {
		ctx.IndentedJSON(http.StatusCreated, gin.H{
			"chat": nil,
		})
		return
	}

	_, err = rds.LRange(redis_ctx, fmt.Sprintf("%d-%d", akun.ID, dataChat.Karakter.ID), 0, -1).Result()
	if err == nil {
		val, _ := rds.LPush(redis_ctx, fmt.Sprintf("%d-%d", akun.ID, dataChat.Karakter.ID), fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])).Result()
		fmt.Println("REDIS TAMBAHIN VAL", val)
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

	val, err := rds.LRange(redis_ctx, fmt.Sprintf("%d-%d", akun.ID, dataChat.Karakter.ID), 0, -1).Result()
	if err == nil {
		fmt.Println("REDIS UPDATE CHAT", val, fmt.Sprintf("%d-%d", akun.ID, dataChat.Karakter.ID))
	}

	ctx.IndentedJSON(http.StatusCreated, gin.H{
		"chat":     resp.Candidates[0].Content.Parts[0],
		"chatDulu": ChatDulu,
		"id":       strconv.FormatUint(isiCharDiPilih.ID, 10),
	})
}

func HapusPesanHandler(ctx *gin.Context) {
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
}

func UpdatePesanHandler(ctx *gin.Context) {
	db, akun := routes.InitAkunDB(ctx)
	dataChat, err := routes.InitChat(ctx)

	if err != nil || len(dataChat.KarakterChat.History) <= 0 {
		ctx.IndentedJSON(http.StatusNotFound, gin.H{
			"status": "error",
		})

		return
	}

	isiChat := models.IsiChat{}
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
}

func SaranKalimatHandler(ctx *gin.Context) {
	_, akun := routes.InitAkunDB(ctx)
	dataChat, err := routes.InitChat(ctx)

	if err != nil {
		ctx.IndentedJSON(http.StatusNotFound, gin.H{
			"chat": nil,
		})

		return
	}

	genAIHistoryChat, _ := chat.DapatinHistoryKarakter(dataChat.KarakterChat)

	cs := chat.BuatChat(client, dataChat.Karakter, akun.Personalitas.DefaultPersonalitas(akun.Username), genAIHistoryChat)

	resp := chat.SaranKalimat(client, akun.Personalitas.DefaultPersonalitas(akun.Username), cs)
	if resp == nil {
		ctx.IndentedJSON(http.StatusCreated, gin.H{
			"chat": nil,
		})

		return
	}

	ctx.IndentedJSON(http.StatusCreated, gin.H{
		"chat": resp.Candidates[0].Content.Parts[0],
	})
}

func PersonalitasHandler(ctx *gin.Context) {
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
	akun := akunRaw.(models.Akun)

	if newPersonalitas.Status == "pilih" {
		checkPersonalitas := models.Personalitas{}
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
		db.Model(&akun).Update("Personalitas", models.Personalitas{ID: checkPersonalitas.ID})
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

	personalitas := models.Personalitas{
		Nama:         newPersonalitas.Nama,
		Personalitas: newPersonalitas.Personalitas,
		AkunID:       akun.ID,
	}

	fmt.Println("STATUS: " + newPersonalitas.Status)
	fmt.Println("ID!!!: " + newPersonalitas.ID)

	if newPersonalitas.Status == "buat" {
		db.Create(&personalitas)
	} else if newPersonalitas.Status == "edit" {
		checkPersonalitas := models.Personalitas{}
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
}

func RegisterHandler(ctx *gin.Context) {
	session := sessions.Default(ctx)
	username := ctx.PostForm("username")
	email := ctx.PostForm("email")
	password := ctx.PostForm("password")

	if strings.Trim(email, " ") == "" || strings.Trim(password, " ") == "" || strings.Trim(username, " ") == "" {
		ctx.Redirect(http.StatusMovedPermanently, "/register")
		return
	}

	var akunDiMasuin models.Akun
	db.First(&akunDiMasuin, "email = ? OR username = ?", email, username)

	if akunDiMasuin.Email != "" || akunDiMasuin.Username != "" {
		if akunDiMasuin.Email != "" {
			session.Set("flash", "Email has already been used!")
		} else {
			session.Set("flash", "Username has already been used!")
		}

		session.Save()

		ctx.Redirect(http.StatusMovedPermanently, "/register")
		return
	}

	hashPassword, err := auth.HashPassword(password)
	if err != nil {
		ctx.Redirect(http.StatusMovedPermanently, "/register")
		return
	}

	registerAkun := models.Akun{Username: username, Email: email, Password: hashPassword, ImageURL: "/assets/no-users.png"}

	db.Create(&registerAkun)

	if err := session.Save(); err != nil {
		fmt.Println(err)
		ctx.Redirect(http.StatusFound, "/register")
		return
	}

	ctx.Redirect(http.StatusFound, "/login")
}

func LoginHandler(ctx *gin.Context) {
	session := sessions.Default(ctx)
	nama_akun := ctx.PostForm("nama_akun")
	password := ctx.PostForm("password")

	var akun models.Akun
	if err := db.Where("email = ? OR username = ?", nama_akun, nama_akun).First(&akun); err.Error != nil {
		session.Set("flash", "Email or password is incorrect")
		session.Save()
		fmt.Println("ERRRORR", err.Error)
		ctx.Redirect(http.StatusMovedPermanently, "/login")
		return
	}

	if !auth.CheckPasswordHash(password, akun.Password) {
		session.Set("flash", "Email or password is incorrect")
		session.Save()
		fmt.Println("PASSWORD SALAH")

		ctx.Redirect(http.StatusMovedPermanently, "/login")
		return
	}

	// if email != "tester144@gmail.com" || password != "123456" {
	// 	ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
	// 	return
	// }

	fmt.Println("SUKSES")

	session.Set("user", akun.ID)
	session.Save()

	ctx.Redirect(http.StatusFound, "/")
}

func LogoutHandler(ctx *gin.Context) {
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
}

func main() {
	database.Connect()
	database.InitCloudinary()
	db, rds = database.GetDatabase()

	client = chat.ClientGenAI()
	defer client.Close()

	r := gin.Default()

	r.Use(sessions.Sessions("session", cookie.NewStore(database.SecretKey)))
	r.SetFuncMap(template.FuncMap{
		"PanjangArrayKurangSatu": utils.PanjangArrayKurangSatu,
	})
	r.Static("/assets", "./assets")
	r.LoadHTMLGlob("templates/*")
	r.StaticFile("/favicon.ico", "./assets/IconPer.png")
	// r.Use(sessions.Sessions("session", cookie.NewStore(secret)))
	// r.MaxMultipartMemory = 20 << 20

	r.GET("/", IndexPage)
	r.GET("/search", SearchPage)
	r.GET("/profile", ProfilePage)

	r.POST("/logout", LogoutHandler)

	redirectLoginAutentikasi := r.Group("/")
	redirectLoginAutentikasi.Use(routes.DapatinAkun())
	redirectLoginAutentikasi.Use(routes.CheckAutentikasi("akses"))
	{
		redirectLoginAutentikasi.GET("/buatkarakter/", BuatKarakterPage)
		redirectLoginAutentikasi.GET("/editkarakter/:idkarakter", EditKarakterPage)
		redirectLoginAutentikasi.GET("/chat/:idkarakter/*idchat", KarakterChatPage)

		redirectLoginAutentikasi.POST("/karakter/", KarakterHandler)

		redirectLoginAutentikasi.POST("/api/chat/", ChatHandler)
		redirectLoginAutentikasi.POST("/api/chat/buatchat", BuatChatHandler)
		redirectLoginAutentikasi.POST("/api/chat/hapuschat", HapusChatHandler)

		redirectLoginAutentikasi.POST("/api/chat/sarankalimat", SaranKalimatHandler)

		redirectLoginAutentikasi.POST("/api/chat/ulangipesan", UlangiPesanHandler)
		redirectLoginAutentikasi.POST("/api/chat/hapuspesan", HapusPesanHandler)
		redirectLoginAutentikasi.POST("/api/chat/updatepesan", UpdatePesanHandler)

		redirectLoginAutentikasi.POST("/personalitas", PersonalitasHandler)
	}

	redirectHomeAutentikasi := r.Group("/")
	redirectHomeAutentikasi.Use(routes.CheckAutentikasi("login"))
	{
		redirectHomeAutentikasi.GET("/login", LoginPage)
		redirectHomeAutentikasi.POST("/login", LoginHandler)

		redirectHomeAutentikasi.GET("/register", RegisterPage)
		redirectHomeAutentikasi.POST("/register", RegisterHandler)
	}

	r.Run()
}
