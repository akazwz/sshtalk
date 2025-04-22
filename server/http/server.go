package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Start 启动HTTP服务器
func Start() {
	openaiBaseURL := os.Getenv("OPENAI_BASE_URL")
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	openaiModel := os.Getenv("OPENAI_MODEL")
	port := os.Getenv("PORT")

	log.Printf("Starting server with OpenAI config - BaseURL: %s, Model: %s", openaiBaseURL, openaiModel)

	openaiClient := openai.NewClient(option.WithBaseURL(openaiBaseURL), option.WithAPIKey(openaiAPIKey))

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/teapot", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("I'm a teapot"))
	})

	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		var data []Message
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		messages := make([]openai.ChatCompletionMessageParamUnion, 0)
		for _, msg := range data {
			switch msg.Role {
			case "user":
				messages = append(messages, openai.UserMessage(msg.Content))
			case "assistant":
				messages = append(messages, openai.AssistantMessage(msg.Content))
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()

		stream := openaiClient.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
			Model:    openaiModel,
			Messages: messages,
		})

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		acc := openai.ChatCompletionAccumulator{}
		for stream.Next() {
			chunk := stream.Current()
			acc.AddChunk(chunk)
			if len(chunk.Choices) > 0 {
				if content := chunk.Choices[0].Delta.Content; content != "" {
					if _, err := w.Write([]byte(content)); err != nil {
						log.Printf("Error writing response: %v", err)
						return
					}
					flusher.Flush()
				}
			}
		}

		if err := stream.Err(); err != nil {
			log.Printf("Stream error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	// Frontend handling
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		isDev := os.Getenv("ENV") == "development"
		if isDev {
			proxy := httputil.NewSingleHostReverseProxy(&url.URL{
				Scheme: "http",
				Host:   "localhost:5173",
			})
			proxy.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	})

	// Create server with timeouts
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	log.Printf("Server starting on port %s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
