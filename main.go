package main

import (
	"bytes"
	"encoding/json"
	"errors"
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

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model       string        `json:"model"`
	Messages    []groqMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type groqResponse struct {
	Choices []struct {
		Message groqMessage `json:"message"`
	} `json:"choices"`
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

	if apiKey := os.Getenv("GROQ_API_KEY"); apiKey != "" {
		reply, err := askGroq(apiKey, input.Message)
		if err == nil {
			json.NewEncoder(w).Encode(chatResponse{Reply: reply})
			return
		}
		json.NewEncoder(w).Encode(chatResponse{Reply: "The AI service is temporarily unavailable. Please try again in a moment."})
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

func askGroq(apiKey, message string) (string, error) {
	model := os.Getenv("GROQ_MODEL")
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}
	payload, err := json.Marshal(groqRequest{
		Model: model,
		Messages: []groqMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: message},
		},
		Temperature: 0.7,
	})
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest(http.MethodPost, "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 45 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var result groqResponse
	if response.StatusCode >= 300 || json.NewDecoder(response.Body).Decode(&result) != nil || len(result.Choices) == 0 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
		return "", errors.New("invalid Groq response")
	}
	return result.Choices[0].Message.Content, nil
}
