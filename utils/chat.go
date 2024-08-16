package utils

import (
	"context"
	"log"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var ctx context.Context = context.Background()

func ClientGenAI() *genai.Client {
	// Access your API key as an environment variable (see "Set up your API key" above)
	client, err := genai.NewClient(ctx, option.WithAPIKey(DapatinEnvVariable("GEMINI_AI")))
	if err != nil {
		log.Fatal(err)
	}

	return client
}

func BuatModel(client *genai.Client, SystemInstruction *genai.Content) *genai.GenerativeModel {
	// The Gemini 1.5 models are versatile and work with most use cases
	model := client.GenerativeModel("gemini-1.5-flash")
	model.SystemInstruction = SystemInstruction

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

	return model
}

func BuatChat(client *genai.Client, karakter Karakter, personalitas Personalitas, history []*genai.Content) *genai.ChatSession {
	model := BuatModel(client, &genai.Content{
		Parts: []genai.Part{
			genai.Text(karakter.RenderPersonalitas(personalitas.Nama)),
			// genai.Text("Your name is Rudi, You are a man and a barber who runs a barbershop, suddenly a customer named John comes to your shop getting a haircut"),
			// genai.Text("Your name is Sarah, You like playing games and studying and currently you have a boyfriend named John. You are now in his house"),
		},
		Role: "model",
	})

	cs := model.StartChat()
	historyChat := history
	if len(history) <= 0 {
		historyChat = append([]*genai.Content{
			{
				Parts: []genai.Part{
					// genai.Text("*Smiles at him* Hello! Welcome to my barbershop! Please sit on the chair."),
					genai.Text(strings.ReplaceAll(strings.ReplaceAll(karakter.Chat, "{{char}}", karakter.Nama), "{{user}}", personalitas.Nama)),
				},
				Role: "model",
			},
		}, history...)
	}

	cs.History = historyChat

	return cs
}

func UlangiJawaban(cs *genai.ChatSession) (*genai.GenerateContentResponse, error) {
	PesanHistory := DapatinSemuaPesan(cs)
	SebelumHistory := PesanHistory[:len(PesanHistory)-2]

	DialogModel := PesanHistory[len(PesanHistory)-1]
	Omongan := PesanHistory[len(PesanHistory)-2]

	if DialogModel.Role == "user" || Omongan.Role != "user" {
		return nil, nil
	}

	cs.History = SebelumHistory
	return cs.SendMessage(ctx, Omongan.Parts[0])
}

func SaranKalimat(client *genai.Client, personalitas Personalitas, cs *genai.ChatSession) *genai.GenerateContentResponse {
	modelSaran := BuatModel(client, &genai.Content{
		Parts: []genai.Part{
			genai.Text(personalitas.RenderPersonalitas(personalitas.Nama)),
			// genai.Text("You are John who went to the Rudi's Barbershop to get a haircut"),
		},
		Role: "user",
	})

	csSaran := modelSaran.StartChat()

	PesanHistory := DapatinSemuaPesan(cs)
	DialogModel := PesanHistory[len(PesanHistory)-1]

	if DialogModel.Role != "model" {
		return nil
	}

	csSaran.History = PesanHistory

	resp, _ := csSaran.SendMessage(ctx, DialogModel.Parts[0])

	// cs.History = PesanHistory

	return resp
}

func KirimPesan(cs *genai.ChatSession, pesan string) (*genai.GenerateContentResponse, error) {
	return cs.SendMessage(ctx, genai.Text(pesan))
}

func DapatinSemuaPesan(cs *genai.ChatSession) []*genai.Content {
	return cs.History
}
