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
	"noodexx/internal/auth"
	"noodexx/internal/config"
	"noodexx/internal/ingest"
	"noodexx/internal/logging"
	providerpkg "noodexx/internal/provider"
	"noodexx/internal/rag"
	"noodexx/internal/skills"
	"noodexx/internal/store"
	"noodexx/internal/watcher"
)

const version = "1.0.0"

// maskAPIKey masks an API key for display, showing only first 8 and last 4 characters
func maskAPIKey(key string) string {
	if key == "" {
		return "Not set"
	}
	if len(key) <= 12 {
		return "••••••••"
	}
	return key[:8] + "..." + key[len(key)-4:]
}

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

// initAuthProvider initializes the authentication provider based on configuration
func initAuthProvider(authStore auth.Store, cfg *config.Config, logger *logging.Logger) auth.Provider {
	authProvider, err := auth.GetProvider(
		cfg.Auth.Provider,
		authStore,
		cfg.Auth.SessionExpiryDays,
		cfg.Auth.LockoutThreshold,
		cfg.Auth.LockoutDurationMinutes,
	)
	if err != nil {
		logger.Error("Failed to initialize auth provider: %v", err)
		log.Fatalf("Failed to initialize auth provider: %v", err)
	}
	logger.Info("Auth provider initialized: %s", cfg.Auth.Provider)
	return authProvider
}

func main() {
	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("=== Configuration Loaded ===")
	log.Printf("User Mode: %s", cfg.UserMode)
	log.Printf("Auth Provider: %s", cfg.Auth.Provider)

	// ANSI color codes
	const (
		colorReset = "\033[0m"
		colorGreen = "\033[32m"
		colorRed   = "\033[31m"
	)

	// Display dual provider configuration
	log.Printf("--- AI Providers ---")
	if cfg.LocalProvider.Type != "" {
		log.Printf("%s🟢%s Local Provider: %s", colorGreen, colorReset, cfg.LocalProvider.Type)
		if cfg.LocalProvider.Type == "ollama" {
			log.Printf("  Endpoint: %s", cfg.LocalProvider.OllamaEndpoint)
			log.Printf("  Chat Model: %s", cfg.LocalProvider.OllamaChatModel)
			log.Printf("  Embed Model: %s", cfg.LocalProvider.OllamaEmbedModel)
		}
	} else {
		log.Printf("Local Provider: Not configured")
	}

	if cfg.CloudProvider.Type != "" {
		log.Printf("%s🔴%s Cloud Provider: %s", colorRed, colorReset, cfg.CloudProvider.Type)
		if cfg.CloudProvider.Type == "openai" {
			log.Printf("  Chat Model: %s", cfg.CloudProvider.OpenAIChatModel)
			log.Printf("  Embed Model: %s", cfg.CloudProvider.OpenAIEmbedModel)
			log.Printf("  API Key: %s", maskAPIKey(cfg.CloudProvider.OpenAIKey))
		} else if cfg.CloudProvider.Type == "anthropic" {
			log.Printf("  Chat Model: %s", cfg.CloudProvider.AnthropicChatModel)
			log.Printf("  Embed Model: %s", cfg.CloudProvider.AnthropicEmbedModel)
			log.Printf("  API Key: %s", maskAPIKey(cfg.CloudProvider.AnthropicKey))
		}
	} else {
		log.Printf("Cloud Provider: Not configured")
	}

	log.Printf("--- Privacy Settings ---")
	if cfg.Privacy.DefaultToLocal {
		log.Printf("Default Provider: %s🟢%s Local AI", colorGreen, colorReset)
	} else {
		log.Printf("Default Provider: %s🔴%s Cloud AI", colorRed, colorReset)
	}
	log.Printf("Cloud RAG Policy: %s", cfg.Privacy.CloudRAGPolicy)
	log.Printf("=============================")

	// Initialize logger
	logger, logWriter, err := initializeLogging(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}
	logger.Info("Starting Noodexx v%s...", version)

	// Initialize store with migrations
	st, err := store.NewStore("noodexx.db", cfg.UserMode)
	if err != nil {
		logger.Error("Failed to initialize store: %v", err)
		os.Exit(1)
	}
	defer st.Close()
	logger.Info("Database initialized")

	// Initialize dual provider manager and RAG policy enforcer
	dualProviderManager, err := providerpkg.NewDualProviderManager(cfg, logger)
	if err != nil {
		logger.Error("Failed to initialize provider manager: %v", err)
		os.Exit(1)
	}
	ragEnforcer := rag.NewRAGPolicyEnforcer(cfg, logger)
	logger.Info("Dual provider manager initialized")

	// Get active provider for backward compatibility with ingester
	provider, err := dualProviderManager.GetActiveProvider()
	if err != nil {
		logger.Error("Failed to get active provider: %v", err)
		os.Exit(1)
	}

	// Initialize RAG components
	chunker := rag.NewChunker(500, 50)
	ragLogger := logging.NewLogger("rag", logging.ParseLevel(cfg.Logging.Level), logWriter)
	searcher := rag.NewSearcher(&storeAdapter{store: st}, ragLogger)
	logger.Info("RAG components initialized")

	// Initialize ingester
	ingestLogger := logging.NewLogger("ingest", logging.ParseLevel(cfg.Logging.Level), logWriter)
	ingester := ingest.NewIngester(&providerAdapter{provider: provider}, st, chunker, false, cfg.Guardrails.AutoSummarize, ingestLogger)
	logger.Info("Ingester initialized")

	// Initialize skills with store adapter for user-scoped loading
	skillsLogger := logging.NewLogger("skills", logging.ParseLevel(cfg.Logging.Level), logWriter)
	skillsStoreAdapter := &skillsStoreAdapter{store: st}
	skillsLoader := skills.NewLoaderWithStore("skills", false, skillsLogger, skillsStoreAdapter)
	loadedSkills, err := skillsLoader.LoadAll()
	if err != nil {
		logger.Warn("Failed to load skills: %v", err)
	} else {
		logger.Info("Loaded %d skills", len(loadedSkills))
	}
	skillsExecutor := skills.NewExecutor(false, skillsLogger)

	// Initialize folder watcher with adapter
	watcherLogger := logging.NewLogger("watcher", logging.ParseLevel(cfg.Logging.Level), logWriter)
	watcherStore := &watcherStoreAdapter{store: st}
	w, err := watcher.NewWatcher(ingester, watcherStore, false, watcherLogger)
	if err != nil {
		logger.Error("Failed to initialize watcher: %v", err)
		os.Exit(1)
	}
	ctx := context.Background()

	// Get local-default user for backward compatibility with config-based folders
	localDefaultUser, err := st.GetUserByUsername(ctx, "local-default")
	if err != nil {
		logger.Warn("Failed to get local-default user: %v", err)
	} else {
		// Add folders from config to local-default user
		for _, folder := range cfg.Folders {
			if err := w.AddFolder(ctx, localDefaultUser.ID, folder); err != nil {
				logger.Warn("Failed to add watched folder %s: %v", folder, err)
			} else {
				logger.Info("Watching folder: %s", folder)
			}
		}
	}
	go w.Start(ctx)

	// Initialize API server with adapters
	apiConfig := &api.ServerConfig{
		PrivacyMode:        false,
		UserMode:           cfg.UserMode,
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

	// Initialize auth provider
	authLogger := logging.NewLogger("auth", logging.ParseLevel(cfg.Logging.Level), logWriter)
	authStoreAdapter := &authStoreAdapter{store: st}
	authProvider := &apiAuthProviderAdapter{
		provider: initAuthProvider(authStoreAdapter, cfg, authLogger),
	}

	// Create adapters for the new components (using already initialized dualProviderManager and ragEnforcer)
	apiProviderManagerAdapter := &apiProviderManagerAdapter{manager: dualProviderManager}
	apiRAGEnforcerAdapter := &apiRAGEnforcerAdapter{enforcer: ragEnforcer}

	apiServer, err := api.NewServer(
		apiStoreAdapter,
		apiProviderAdapter,
		ingester,
		apiSearcherAdapter,
		apiConfig,
		apiSkillsLoaderAdapter,
		apiSkillsExecutorAdapter,
		apiLoggerAdapter,
		authProvider,
		"config.json",
		apiProviderManagerAdapter,
		apiRAGEnforcerAdapter,
	)
	if err != nil {
		logger.Error("Failed to initialize API server: %v", err)
		os.Exit(1)
	}
	logger.Info("API server initialized")

	// Register routes
	mux := http.NewServeMux()
	apiServer.RegisterRoutes(mux)

	// Apply authentication middleware
	authMiddleware := auth.AuthMiddleware(authStoreAdapter, cfg.UserMode)
	handler := authMiddleware(mux)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.BindAddress, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on http://%s", addr)
		log.Printf("Press Ctrl-C to quit")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error: %v", err)
		}
	}()

	// Start background job for token cleanup
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		logger.Info("Token cleanup job started (runs every hour)")

		for range ticker.C {
			ctx := context.Background()
			if err := st.CleanupExpiredTokens(ctx); err != nil {
				logger.Error("Failed to cleanup expired tokens: %v", err)
			} else {
				logger.Debug("Expired tokens cleaned up")
			}
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
