package main

import (
	"html/template"
	"net/http"

	"strings"

	"persona/database"
	"persona/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
)

type CHAT struct {
	Id   string `json:"id"`
	Chat string `json:"chat"`
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
			akun := database.Akun{}

			db.First(&akun, user)

			if akun.ID == 0 {
				session.Delete("user")
			}

			session.Save()
			return
		}

		if status == "login" && user != nil {
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

	r.GET("/", func(ctx *gin.Context) {
		db := database.GetDatabase()
		session := sessions.Default(ctx)
		user := session.Get("user")

		var akun database.Akun
		db.First(&akun, user)

		ctx.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Main",
			"akun":  akun,
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

			var akun database.Akun
			db.First(&akun, user)

			ctx.HTML(http.StatusOK, "buatkarakter.html", gin.H{
				"title": "Create Character",
				"akun":  akun,
			})
		})

		HistoryChat := []*genai.Content{}
		redirectLoginAutentikasi.GET("/chat/:idchat", func(c *gin.Context) {
			cs := utils.BuatChat(client, HistoryChat)

			if len(HistoryChat) <= 0 {
				HistoryChat = utils.DapatinSemuaPesan(cs)
			}

			c.HTML(http.StatusOK, "chat.html", gin.H{
				"title":       "Chat",
				"idChat":      c.Param("idchat"),
				"isi":         HistoryChat,
				"PanjangChat": len(HistoryChat) - 1,
			})
		})

		redirectLoginAutentikasi.POST("/chat", func(c *gin.Context) {
			cs := utils.BuatChat(client, HistoryChat)

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

		redirectLoginAutentikasi.POST("/chat/ulangipesan", func(ctx *gin.Context) {
			cs := utils.BuatChat(client, HistoryChat)

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

		redirectLoginAutentikasi.POST("/chat/sarankalimat", func(ctx *gin.Context) {
			cs := utils.BuatChat(client, HistoryChat)

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
			akun := database.Akun{}

			db.First(&akun, "email = ? OR username = ?", nama_akun, nama_akun)

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
			session.Set("user", akun.ID) // In real world usage you'd set this to the users ID
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

			akunDiMasuin := database.Akun{}
			db.First(&akunDiMasuin, "email = ? OR username = ?", email, username)

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

			registerAkun := database.Akun{Username: username, Email: email, Password: hashPassword, ImageURL: "/assets/no-users.png"}

			db.Create(&registerAkun)

			session.Set("user", registerAkun.ID)
			if err := session.Save(); err != nil {
				ctx.Redirect(http.StatusFound, "/register")
				return
			}

			ctx.Redirect(http.StatusFound, "/")
		})
	}

	r.Run()
}
