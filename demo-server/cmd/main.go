package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aspectrr/fluid.sh/demo-server/internal/ws"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if present (won't override existing env vars)
	_ = godotenv.Load()

	port := flag.String("port", "8090", "HTTP server port")
	fluidBin := flag.String("fluid-bin", "fluid", "Path to fluid binary")
	allowedOrigins := flag.String("allowed-origins", "*", "Comma-separated allowed origins")
	sessionTimeout := flag.Duration("session-timeout", 10*time.Minute, "Session inactivity timeout")
	maxSessions := flag.Int("max-sessions", 20, "Maximum concurrent sessions")
	llmAPIKey := flag.String("llm-api-key", "", "OpenRouter API key")
	llmModel := flag.String("llm-model", "openai/gpt-oss-120b:free", "LLM model to use")
	flag.Parse()

	if *llmAPIKey == "" {
		*llmAPIKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if *llmAPIKey == "" {
		log.Fatal("--llm-api-key or OPENROUTER_API_KEY required")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	origins := strings.Split(*allowedOrigins, ",")

	handler := ws.NewHandler(ws.Config{
		FluidBin:       *fluidBin,
		AllowedOrigins: origins,
		SessionTimeout: *sessionTimeout,
		MaxSessions:    *maxSessions,
		LLMAPIKey:      *llmAPIKey,
		LLMModel:       *llmModel,
		Logger:         logger,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws/demo", handler.HandleWebSocket)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:    ":" + *port,
		Handler: corsMiddleware(mux, origins),
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("shutting down")
		handler.Shutdown()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	logger.Info("starting demo server", "port", *port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler, origins []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		for _, o := range origins {
			if o == "*" || o == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
