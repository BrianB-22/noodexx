package main

import (
	"context"
	"fmt"
	"io"
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

// initializeLogging creates and configures the logger based on configuration
func initializeLogging(cfg *config.Config) (*logging.Logger, io.Writer, error) {
	var writer io.Writer

	if cfg.Logging.DebugEnabled {
		// Create file writer with rotation
		fileWriter, err := logging.NewFileWriter(
			cfg.Logging.File,
			cfg.Logging.MaxSizeMB,
			cfg.Logging.MaxBackups,
		)
		if err != nil {
			// Log error to stderr and fall back to console-only
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to create debug log file: %v\n", err)
			fmt.Fprintf(os.Stderr, "[INFO] Falling back to console-only logging\n")
			writer = os.Stdout
		} else {
			// Create multi-writer for dual output (console + file)
			writer = logging.NewMultiWriter(os.Stdout, fileWriter, true)
		}
	} else {
		// Debug disabled: console-only
		writer = os.Stdout
	}

	// Parse log level and create logger
	level := logging.ParseLevel(cfg.Logging.Level)
	return logging.NewLogger("main", level, writer), writer, nil
}

func main() {
	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("=== Configuration Loaded ===")
	log.Printf("Provider Type: %s", cfg.Provider.Type)
	log.Printf("Ollama Chat Model: %s", cfg.Provider.OllamaChatModel)
	log.Printf("Ollama Embed Model: %s", cfg.Provider.OllamaEmbedModel)
	log.Printf("=============================")

	// Initialize logger
	logger, logWriter, err := initializeLogging(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}
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
	llmLogger := logging.NewLogger("llm", logging.ParseLevel(cfg.Logging.Level), logWriter)
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
	}, cfg.Privacy.Enabled, llmLogger)
	if err != nil {
		logger.Error("Failed to initialize LLM provider: %v", err)
		os.Exit(1)
	}
	logger.Info("LLM provider: %s (privacy mode: %v)", provider.Name(), cfg.Privacy.Enabled)

	// Initialize RAG components
	chunker := rag.NewChunker(500, 50)
	ragLogger := logging.NewLogger("rag", logging.ParseLevel(cfg.Logging.Level), logWriter)
	searcher := rag.NewSearcher(&storeAdapter{store: st}, ragLogger)
	logger.Info("RAG components initialized")

	// Initialize ingester
	ingestLogger := logging.NewLogger("ingest", logging.ParseLevel(cfg.Logging.Level), logWriter)
	ingester := ingest.NewIngester(&providerAdapter{provider: provider}, st, chunker, cfg.Privacy.Enabled, cfg.Guardrails.AutoSummarize, ingestLogger)
	logger.Info("Ingester initialized")

	// Initialize skills
	skillsLogger := logging.NewLogger("skills", logging.ParseLevel(cfg.Logging.Level), logWriter)
	skillsLoader := skills.NewLoader("skills", cfg.Privacy.Enabled, skillsLogger)
	loadedSkills, err := skillsLoader.LoadAll()
	if err != nil {
		logger.Warn("Failed to load skills: %v", err)
	} else {
		logger.Info("Loaded %d skills", len(loadedSkills))
	}
	skillsExecutor := skills.NewExecutor(cfg.Privacy.Enabled, skillsLogger)

	// Initialize folder watcher with adapter
	watcherLogger := logging.NewLogger("watcher", logging.ParseLevel(cfg.Logging.Level), logWriter)
	watcherStore := &watcherStoreAdapter{store: st}
	w, err := watcher.NewWatcher(ingester, watcherStore, cfg.Privacy.Enabled, watcherLogger)
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
		PrivacyMode:        cfg.Privacy.Enabled,
		Provider:           cfg.Provider.Type,
		OllamaEndpoint:     cfg.Provider.OllamaEndpoint,
		OllamaEmbedModel:   cfg.Provider.OllamaEmbedModel,
		OllamaChatModel:    cfg.Provider.OllamaChatModel,
		OpenAIKey:          cfg.Provider.OpenAIKey,
		OpenAIEmbedModel:   cfg.Provider.OpenAIEmbedModel,
		OpenAIChatModel:    cfg.Provider.OpenAIChatModel,
		AnthropicKey:       cfg.Provider.AnthropicKey,
		AnthropicChatModel: cfg.Provider.AnthropicChatModel,
	}
	apiStoreAdapter := &apiStoreAdapter{store: st}
	apiProviderAdapter := &apiProviderAdapter{provider: provider}
	apiSearcherAdapter := &apiSearcherAdapter{searcher: searcher}
	apiSkillsLoaderAdapter := &apiSkillsLoaderAdapter{loader: skillsLoader}
	apiSkillsExecutorAdapter := &apiSkillsExecutorAdapter{executor: skillsExecutor}
	apiLoggerAdapter := &apiLoggerAdapter{logger: logger}

	apiServer, err := api.NewServer(
		apiStoreAdapter,
		apiProviderAdapter,
		ingester,
		apiSearcherAdapter,
		apiConfig,
		apiSkillsLoaderAdapter,
		apiSkillsExecutorAdapter,
		apiLoggerAdapter,
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
