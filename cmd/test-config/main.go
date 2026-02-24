package main

import (
	"fmt"
	"log"
	"noodexx/internal/config"
)

func main() {
	// Test loading config
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("Configuration loaded successfully!")
	fmt.Printf("Provider: %s\n", cfg.Provider.Type)
	fmt.Printf("Privacy Mode: %v\n", cfg.Privacy.Enabled)
	fmt.Printf("Server Port: %d\n", cfg.Server.Port)
	fmt.Printf("Server Bind Address: %s\n", cfg.Server.BindAddress)
	fmt.Printf("Log Level: %s\n", cfg.Logging.Level)
	fmt.Printf("Max File Size: %d MB\n", cfg.Guardrails.MaxFileSizeMB)
	fmt.Printf("PII Detection: %s\n", cfg.Guardrails.PIIDetection)
}
