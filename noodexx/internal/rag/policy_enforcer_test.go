package rag

import (
	"bytes"
	"noodexx/internal/config"
	"noodexx/internal/logging"
	"testing"
)

// Helper function to create a test logger
func createTestLogger() *logging.Logger {
	buf := &bytes.Buffer{}
	return logging.NewLogger("test", logging.DEBUG, buf)
}

// Helper function to create a test config with specified settings
func createTestConfig(useLocalAI bool, cloudRAGPolicy string) *config.Config {
	return &config.Config{
		Privacy: config.PrivacyConfig{
			DefaultToLocal:     useLocalAI,
			CloudRAGPolicy: cloudRAGPolicy,
		},
	}
}

func TestNewRAGPolicyEnforcer(t *testing.T) {
	cfg := createTestConfig(true, "no_rag")
	logger := createTestLogger()

	enforcer := NewRAGPolicyEnforcer(cfg, logger)

	if enforcer == nil {
		t.Fatal("NewRAGPolicyEnforcer returned nil")
	}
	if enforcer.config != cfg {
		t.Error("enforcer config not set correctly")
	}
	if enforcer.logger != logger {
		t.Error("enforcer logger not set correctly")
	}
}

func TestShouldPerformRAG_LocalMode(t *testing.T) {
	tests := []struct {
		name           string
		cloudRAGPolicy string
		expected       bool
	}{
		{
			name:           "local mode with no_rag policy",
			cloudRAGPolicy: "no_rag",
			expected:       true, // RAG always enabled in local mode
		},
		{
			name:           "local mode with allow_rag policy",
			cloudRAGPolicy: "allow_rag",
			expected:       true, // RAG always enabled in local mode
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig(true, tt.cloudRAGPolicy)
			logger := createTestLogger()
			enforcer := NewRAGPolicyEnforcer(cfg, logger)

			result := enforcer.ShouldPerformRAG()

			if result != tt.expected {
				t.Errorf("ShouldPerformRAG() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldPerformRAG_CloudMode(t *testing.T) {
	tests := []struct {
		name           string
		cloudRAGPolicy string
		expected       bool
	}{
		{
			name:           "cloud mode with no_rag policy",
			cloudRAGPolicy: "no_rag",
			expected:       false, // RAG disabled
		},
		{
			name:           "cloud mode with allow_rag policy",
			cloudRAGPolicy: "allow_rag",
			expected:       true, // RAG enabled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig(false, tt.cloudRAGPolicy)
			logger := createTestLogger()
			enforcer := NewRAGPolicyEnforcer(cfg, logger)

			result := enforcer.ShouldPerformRAG()

			if result != tt.expected {
				t.Errorf("ShouldPerformRAG() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetRAGStatus_LocalMode(t *testing.T) {
	tests := []struct {
		name           string
		cloudRAGPolicy string
		expected       string
	}{
		{
			name:           "local mode with no_rag policy",
			cloudRAGPolicy: "no_rag",
			expected:       "RAG Enabled (Local)",
		},
		{
			name:           "local mode with allow_rag policy",
			cloudRAGPolicy: "allow_rag",
			expected:       "RAG Enabled (Local)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig(true, tt.cloudRAGPolicy)
			logger := createTestLogger()
			enforcer := NewRAGPolicyEnforcer(cfg, logger)

			result := enforcer.GetRAGStatus()

			if result != tt.expected {
				t.Errorf("GetRAGStatus() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetRAGStatus_CloudMode(t *testing.T) {
	tests := []struct {
		name           string
		cloudRAGPolicy string
		expected       string
	}{
		{
			name:           "cloud mode with no_rag policy",
			cloudRAGPolicy: "no_rag",
			expected:       "RAG Disabled (Cloud Policy)",
		},
		{
			name:           "cloud mode with allow_rag policy",
			cloudRAGPolicy: "allow_rag",
			expected:       "RAG Enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig(false, tt.cloudRAGPolicy)
			logger := createTestLogger()
			enforcer := NewRAGPolicyEnforcer(cfg, logger)

			result := enforcer.GetRAGStatus()

			if result != tt.expected {
				t.Errorf("GetRAGStatus() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestShouldPerformRAG_AllCombinations(t *testing.T) {
	tests := []struct {
		name           string
		useLocalAI     bool
		cloudRAGPolicy string
		expected       bool
	}{
		{
			name:           "local AI + no_rag",
			useLocalAI:     true,
			cloudRAGPolicy: "no_rag",
			expected:       true,
		},
		{
			name:           "local AI + allow_rag",
			useLocalAI:     true,
			cloudRAGPolicy: "allow_rag",
			expected:       true,
		},
		{
			name:           "cloud AI + no_rag",
			useLocalAI:     false,
			cloudRAGPolicy: "no_rag",
			expected:       false,
		},
		{
			name:           "cloud AI + allow_rag",
			useLocalAI:     false,
			cloudRAGPolicy: "allow_rag",
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig(tt.useLocalAI, tt.cloudRAGPolicy)
			logger := createTestLogger()
			enforcer := NewRAGPolicyEnforcer(cfg, logger)

			result := enforcer.ShouldPerformRAG()

			if result != tt.expected {
				t.Errorf("ShouldPerformRAG() = %v, want %v", result, tt.expected)
			}
		})
	}
}
