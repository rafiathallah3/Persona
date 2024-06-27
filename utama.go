package main

import (
	// "context"
	// "fmt"
	"context"
	"fmt"
	"net/http"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type CHAT struct {
	Chat string `json:"chat"`
}

func main() {
	//sk-proj-pSehnOF8HPuzFkEU2NK0T3BlbkFJxrUc1Pu0PXkHruHhzjbN

	ctx := context.Background()
	// // Access your API key as an environment variable (see "Set up your API key" above)
	client, err := genai.NewClient(ctx, option.WithAPIKey("AIzaSyBkb0d6bCmB4Zc38-kq7JcG7CphgnXahQk"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// The Gemini 1.5 models are versatile and work with most use cases
	model := client.GenerativeModel("gemini-1.5-flash")
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text("Your name is Sarah, You like playing games and studying and currently you have a boyfriend named John. You are now in his house and cuddles with him"),
		},
		Role: "model",
	}

	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockNone,
		},
		// {
		// 	Category:  genai.HarmCategoryDangerous,
		// 	Threshold: genai.HarmBlockNone,
		// },
	}

	cs := model.StartChat()
	cs.History = []*genai.Content{
		{
			Parts: []genai.Part{
				genai.Text("*Hugging him tightly* It feels really nice to have you here, John."),
			},
			Role: "model",
		},
	}

	r := gin.Default()
	r.Static("/assets", "./assets")
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Main",
		})
	})

	r.GET("/profile", func(c *gin.Context) {
		c.HTML(http.StatusOK, "profile.html", gin.H{
			"title": "Profile",
		})
	})

	r.POST("/chat", func(c *gin.Context) {
		fmt.Println("POST")
		var newChat CHAT

		if err := c.BindJSON(&newChat); err != nil {
			fmt.Println(err)
			return
		}

		resp, err := cs.SendMessage(ctx, genai.Text(newChat.Chat))
		if err != nil {
			log.Fatal(err)
		}

		c.IndentedJSON(http.StatusCreated, gin.H{
			"chat": resp.Candidates[0].Content.Parts[0],
		})
	})

	r.Run()

	// // inputText := genai.Text(input)

	// // cs.History = append(cs.History, &genai.Content{
	// // 	Parts: []genai.Part{
	// // 		genai.Text(genai.Text(input)),
	// // 	},
	// // 	Role: "user",
	// // })

	// // cs.History = append(cs.History, &genai.Content{
	// // 	Parts: resp.Candidates[0].Content.Parts,
	// // 	Role:  resp.Candidates[0].Content.Role,
	// // })

	// fmt.Printf("Her: %s", resp.Candidates[0].Content.Parts[0])
	// // for _, p := range resp.Candidates[0].Content.Parts {
	// // 	break
	// // }
}
