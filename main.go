package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type chatRequest struct {
	Message string `json:"message"`
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

type chatResponse struct {
	Reply string `json:"reply"`
}

const systemPrompt = `You are the friendly AI assistant on Alexey Tsoy's personal website. Always answer in English, briefly and helpfully. You can tell visitors that Alexey is interested in IT and programming, is learning Go, and enjoys music (especially Justin Bieber), TikTok, and walks with friends. His goal is to become a Senior developer and work for a strong IT company in Kazakhstan or the United States. If the question is not about his profile, still help in a friendly and useful way.`

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.HandleFunc("/api/chat", handleChat)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Portfolio is running at http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var input chatRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Message) == "" {
		http.Error(w, `{"error":"Enter a message"}`, http.StatusBadRequest)
		return
	}

	endpoint := os.Getenv("OLLAMA_URL")
	if endpoint == "" {
		endpoint = "http://localhost:11434/api/generate"
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2"
	}

	payload, _ := json.Marshal(ollamaRequest{Model: model, Prompt: systemPrompt + "\n\nVisitor message: " + input.Message, Stream: false})
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(payload))
	if err != nil {
		json.NewEncoder(w).Encode(chatResponse{Reply: "Llama is not connected yet. Start Ollama and run: ollama pull llama3.2"})
		return
	}
	defer resp.Body.Close()

	var result ollamaResponse
	if resp.StatusCode >= 300 || json.NewDecoder(resp.Body).Decode(&result) != nil || strings.TrimSpace(result.Response) == "" {
		json.NewEncoder(w).Encode(chatResponse{Reply: "Could not get a response from Llama. Check that Ollama is running and the model is installed."})
		return
	}
	json.NewEncoder(w).Encode(chatResponse{Reply: result.Response})
}
