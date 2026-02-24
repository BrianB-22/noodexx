package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"noodexx/internal/config"
	"noodexx/internal/ingest"
	"noodexx/internal/llm"
	"noodexx/internal/logging"
	"noodexx/internal/rag"
	"noodexx/internal/store"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := logging.NewLogger("main", logging.ParseLevel(cfg.Logging.Level), nil)
	logger.Info("Starting Noodexx Phase 2...")

	// Initialize store with migrations
	st, err := store.NewStore("noodexx.db")
	if err != nil {
		logger.Error("Failed to initialize store: %v", err)
		os.Exit(1)
	}
	defer st.Close()
	logger.Info("Database initialized with migrations")

	// Initialize LLM provider
	provider, err := llm.NewProvider(llm.Config{
		Type: cfg.Provider.Type, OllamaEndpoint: cfg.Provider.OllamaEndpoint,
		OllamaEmbedModel: cfg.Provider.OllamaEmbedModel, OllamaChatModel: cfg.Provider.OllamaChatModel,
		OpenAIKey: cfg.Provider.OpenAIKey, OpenAIEmbedModel: cfg.Provider.OpenAIEmbedModel,
		OpenAIChatModel: cfg.Provider.OpenAIChatModel, AnthropicKey: cfg.Provider.AnthropicKey,
		AnthropicEmbedModel: cfg.Provider.AnthropicEmbedModel, AnthropicChatModel: cfg.Provider.AnthropicChatModel,
	}, cfg.Privacy.Enabled)
	if err != nil {
		logger.Error("Failed to initialize LLM provider: %v", err)
		os.Exit(1)
	}
	logger.Info("LLM provider initialized: %s (privacy mode: %v)", provider.Name(), cfg.Privacy.Enabled)

	// Initialize RAG components
	chunker := rag.NewChunker(500, 50)
	searcher := rag.NewSearcher(&storeAdapter{store: st})
	promptBuilder := rag.NewPromptBuilder()
	logger.Info("RAG components initialized")

	// Initialize ingester
	ingester := ingest.NewIngester(&providerAdapter{provider: provider}, st, chunker, cfg.Privacy.Enabled, cfg.Guardrails.AutoSummarize)
	logger.Info("Ingester initialized")

	// TODO: Initialize skill loader and folder watcher (not yet implemented)
	// TODO: Initialize API server (api package not yet implemented)

	// Create basic HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Noodexx Phase 2\n\nComponents initialized:\n- Store: ✓\n- LLM Provider: %s ✓\n- RAG: ✓\n- Ingester: ✓\n\nAPI package coming soon...\n", provider.Name())
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.BindAddress, cfg.Server.Port)
	server := &http.Server{Addr: addr, Handler: mux}

	// Start server in goroutine
	go func() {
		logger.Info("HTTP server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error: %v", err)
		}
	}()

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	logger.Info("Noodexx stopped")

	// Suppress unused variable warnings for now
	_, _, _ = searcher, promptBuilder, ingester
}
