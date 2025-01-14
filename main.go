package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

type Payload struct {
	JsonObject string `json:"json_object"`
	Language   string `json:"language"`
}


func renderIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		slog.Error("Failed to parse index.html", slog.Any("error", err))
		http.Error(w, "Failed to parse index", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		slog.Error("Failed to execute index.html", slog.Any("error", err))
		http.Error(w, "An error occured while serving file", http.StatusInternalServerError)
		return
	}
}

func GenerateInterfaces(w http.ResponseWriter, r *http.Request) {
	var payload Payload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if payload.JsonObject == "" || payload.Language == "" {
		http.Error(w, "(json_object, language) fields are required", http.StatusBadRequest)
		return
	}

	resp, err := Gemini(payload.JsonObject, payload.Language)
	if err != nil {
		slog.Error("Failed to generate the interface", slog.Any("error", err))
		http.Error(w, "Failed to generate the interface", http.StatusInternalServerError)
		return
	}

	res := map[string]interface{}{
		"response": resp,
	}

	json.NewEncoder(w).Encode(res)
}

func formatedResponse(resp *genai.GenerateContentResponse) string {
	var response string

	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				d := fmt.Sprintf("%v", part)
				response += d
			}
		}
	}
	return response
}

func Gemini(jsonObject string, lang string) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(
		ctx,
		option.WithAPIKey(os.Getenv("GEMINI_API_KEY")),
	)
	if err != nil {
		slog.Error("Failed to instantiate gemini client", slog.Any("error", err))
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-pro-latest")
	model.ResponseMIMEType = "application/json"
	prompt := fmt.Sprintf(`
	Data: %v 
	Please return a JSON object with the following structure:
	{
		"interface": "the equivalent of the above data represented in %v"
	}
	Make sure to translate or represent the provided data into a valid %v interface.
	`, jsonObject, lang, lang)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		slog.Error("Failed to generate interface", slog.Any("error", err))
		return "", err
	}

	return formatedResponse(resp), nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load .env, %v", err)
	}
	r := http.NewServeMux()

	r.HandleFunc("/", renderIndex)
	r.HandleFunc("/generate", GenerateInterfaces)

	fs := http.FileServer(http.Dir("./"))
	r.Handle("/static/", http.StripPrefix("/static/",fs))

	log.Println("server running...")
	port := fmt.Sprintf(":%v",os.Getenv("PORT"))
	err = http.ListenAndServe(port, r)
	if err != nil {
		log.Fatal(err)
	}
}
