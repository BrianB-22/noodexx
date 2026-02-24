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

	"noodexx/internal/api"
	"noodexx/internal/config"
	"noodexx/internal/ingest"
	"noodexx/internal/llm"
	"noodexx/internal/logging"
	"noodexx/internal/rag"
	"noodexx/internal/skills"
	"noodexx/internal/store"
	"noodexx/internal/watcher"
)

const version = "1.0.0"

func main() {
	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := logging.NewLogger("main", logging.ParseLevel(cfg.Logging.Level), nil)
	logger.Info("Starting Noodexx v%s...", version)

	// Initialize store with migrations
	st, err := store.NewStore("noodexx.db")
	if err != nil {
		logger.Error("Failed to initialize store: %v", err)
		os.Exit(1)
	}
	defer st.Close()
	logger.Info("Database initialized")

	// Initialize LLM provider
	provider, err := llm.NewProvider(llm.Config{
		Type:                cfg.Provider.Type,
		OllamaEndpoint:      cfg.Provider.OllamaEndpoint,
		OllamaEmbedModel:    cfg.Provider.OllamaEmbedModel,
		OllamaChatModel:     cfg.Provider.OllamaChatModel,
		OpenAIKey:           cfg.Provider.OpenAIKey,
		OpenAIEmbedModel:    cfg.Provider.OpenAIEmbedModel,
		OpenAIChatModel:     cfg.Provider.OpenAIChatModel,
		AnthropicKey:        cfg.Provider.AnthropicKey,
		AnthropicEmbedModel: cfg.Provider.AnthropicEmbedModel,
		AnthropicChatModel:  cfg.Provider.AnthropicChatModel,
	}, cfg.Privacy.Enabled)
	if err != nil {
		logger.Error("Failed to initialize LLM provider: %v", err)
		os.Exit(1)
	}
	logger.Info("LLM provider: %s (privacy mode: %v)", provider.Name(), cfg.Privacy.Enabled)

	// Initialize RAG components
	chunker := rag.NewChunker(500, 50)
	searcher := rag.NewSearcher(&storeAdapter{store: st})
	logger.Info("RAG components initialized")

	// Initialize ingester
	ingester := ingest.NewIngester(&providerAdapter{provider: provider}, st, chunker, cfg.Privacy.Enabled, cfg.Guardrails.AutoSummarize)
	logger.Info("Ingester initialized")

	// Initialize skills
	skillsLoader := skills.NewLoader("skills", cfg.Privacy.Enabled)
	loadedSkills, err := skillsLoader.LoadAll()
	if err != nil {
		logger.Warn("Failed to load skills: %v", err)
	} else {
		logger.Info("Loaded %d skills", len(loadedSkills))
	}
	skillsExecutor := skills.NewExecutor(cfg.Privacy.Enabled)

	// Initialize folder watcher with adapter
	watcherStore := &watcherStoreAdapter{store: st}
	w, err := watcher.NewWatcher(ingester, watcherStore, cfg.Privacy.Enabled)
	if err != nil {
		logger.Error("Failed to initialize watcher: %v", err)
		os.Exit(1)
	}
	ctx := context.Background()
	for _, folder := range cfg.Folders {
		if err := w.AddFolder(ctx, folder); err != nil {
			logger.Warn("Failed to add watched folder %s: %v", folder, err)
		} else {
			logger.Info("Watching folder: %s", folder)
		}
	}
	go w.Start(ctx)

	// Initialize API server with adapters
	apiConfig := &api.ServerConfig{
		PrivacyMode: cfg.Privacy.Enabled,
		Provider:    cfg.Provider.Type,
	}
	apiStoreAdapter := &apiStoreAdapter{store: st}
	apiProviderAdapter := &apiProviderAdapter{provider: provider}
	apiSearcherAdapter := &apiSearcherAdapter{searcher: searcher}
	apiSkillsLoaderAdapter := &apiSkillsLoaderAdapter{loader: skillsLoader}
	apiSkillsExecutorAdapter := &apiSkillsExecutorAdapter{executor: skillsExecutor}

	apiServer, err := api.NewServer(
		apiStoreAdapter,
		apiProviderAdapter,
		ingester,
		apiSearcherAdapter,
		apiConfig,
		apiSkillsLoaderAdapter,
		apiSkillsExecutorAdapter,
	)
	if err != nil {
		logger.Error("Failed to initialize API server: %v", err)
		os.Exit(1)
	}
	logger.Info("API server initialized")

	// Register routes
	mux := http.NewServeMux()
	apiServer.RegisterRoutes(mux)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.BindAddress, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server listening on http://%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error: %v", err)
		}
	}()

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	logger.Info("Noodexx stopped")
}
