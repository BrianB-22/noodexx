package rag

import (
	"noodexx/internal/config"
	"noodexx/internal/logging"
)

// RAGPolicyEnforcer determines whether RAG should be performed based on
// the active provider and configured policy. It implements the core privacy
// control logic for document sharing with cloud providers.
type RAGPolicyEnforcer struct {
	config *config.Config
	logger *logging.Logger
}

// NewRAGPolicyEnforcer creates a new RAG policy enforcer
func NewRAGPolicyEnforcer(cfg *config.Config, logger *logging.Logger) *RAGPolicyEnforcer {
	return &RAGPolicyEnforcer{
		config: cfg,
		logger: logger,
	}
}

// ShouldPerformRAG returns true if RAG should be executed for the current request.
// Decision logic:
// - If using local AI (Privacy.UseLocalAI == true): always perform RAG
// - If using cloud AI (Privacy.UseLocalAI == false):
//   - If CloudRAGPolicy == "allow_rag": perform RAG
//   - If CloudRAGPolicy == "no_rag": do NOT perform RAG
func (e *RAGPolicyEnforcer) ShouldPerformRAG() bool {
	// Local AI mode: always perform RAG
	if e.config.Privacy.UseLocalAI {
		e.logger.Debug("RAG enabled: using local AI provider")
		return true
	}

	// Cloud AI mode: check RAG policy
	if e.config.Privacy.CloudRAGPolicy == "allow_rag" {
		e.logger.Debug("RAG enabled: cloud AI with allow_rag policy")
		return true
	}

	e.logger.Debug("RAG disabled: cloud AI with no_rag policy")
	return false
}

// GetRAGStatus returns a human-readable status string for UI display.
// Returns one of:
// - "RAG Enabled (Local)" - when using local AI
// - "RAG Enabled" - when using cloud AI with allow_rag policy
// - "RAG Disabled (Cloud Policy)" - when using cloud AI with no_rag policy
func (e *RAGPolicyEnforcer) GetRAGStatus() string {
	// Local AI mode: always enabled
	if e.config.Privacy.UseLocalAI {
		return "RAG Enabled (Local)"
	}

	// Cloud AI mode: check policy
	if e.config.Privacy.CloudRAGPolicy == "allow_rag" {
		return "RAG Enabled"
	}

	return "RAG Disabled (Cloud Policy)"
}

// Reload updates the enforcer's config reference
// This should be called after configuration changes to ensure the enforcer
// uses the latest privacy settings
func (e *RAGPolicyEnforcer) Reload(cfg interface{}) {
	if c, ok := cfg.(*config.Config); ok {
		e.config = c
		e.logger.Debug("RAG policy enforcer config reloaded")
	}
}
