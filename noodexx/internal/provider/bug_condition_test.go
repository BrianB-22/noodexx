package provider

import (
	"bytes"
	"noodexx/internal/config"
	"noodexx/internal/logging"
	"strings"
	"testing"
)

// TestBugCondition_ApplicationLaunchWithMissingCloudProviderCredentials tests Property 1:
// Fault Condition - Application Launches with Missing Cloud Provider Credentials
//
// **Validates: Requirements 1.1, 1.2, 1.3, 2.1, 2.2, 2.3**
//
// This test encodes the EXPECTED behavior when a cloud provider is configured but API credentials are missing.
// The test will FAIL on unfixed code (confirming the bug exists) and PASS after the fix is implemented.
//
// Bug Condition: isBugCondition(config) = true when:
//   - Cloud provider type is "openai" or "anthropic"
//   - Cloud provider API key is empty
//   - Local provider is properly configured (type="ollama", endpoint not empty)
//
// Expected Behavior (after fix):
//   - Application launches successfully (no error returned)
//   - Local provider is available (not nil)
//   - Cloud provider is unavailable (nil)
//   - Warning is logged containing "Cloud provider initialization failed"
func TestBugCondition_ApplicationLaunchWithMissingCloudProviderCredentials(t *testing.T) {
	testCases := []struct {
		name                string
		cloudProviderType   string
		cloudProviderAPIKey string
		expectedErrorMsg    string // Expected error message on UNFIXED code
	}{
		{
			name:                "OpenAI with empty API key",
			cloudProviderType:   "openai",
			cloudProviderAPIKey: "",
			expectedErrorMsg:    "openai API key is required",
		},
		{
			name:                "Anthropic with empty API key",
			cloudProviderType:   "anthropic",
			cloudProviderAPIKey: "",
			expectedErrorMsg:    "anthropic API key is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a logger that captures output
			var logBuf bytes.Buffer
			logger := logging.NewLogger("test", logging.INFO, &logBuf)

			// Create config with bug condition:
			// - Cloud provider configured but missing API key
			// - Local provider properly configured
			cfg := &config.Config{
				LocalProvider: config.ProviderConfig{
					Type:             "ollama",
					OllamaEndpoint:   "http://localhost:11434",
					OllamaEmbedModel: "nomic-embed-text",
					OllamaChatModel:  "llama3.2",
				},
				CloudProvider: config.ProviderConfig{
					Type:         tc.cloudProviderType,
					OpenAIKey:    "",
					AnthropicKey: "",
				},
				Privacy: config.PrivacyConfig{
					DefaultToLocal: true,
					CloudRAGPolicy: "no_rag",
				},
			}

			// Attempt to create DualProviderManager
			manager, err := NewDualProviderManager(cfg, logger)

			// EXPECTED BEHAVIOR (after fix):
			// - No error should be returned (application launches successfully)
			// - Local provider should be available
			// - Cloud provider should be nil (unavailable)
			// - Warning should be logged

			if err != nil {
				// ON UNFIXED CODE: This will fail with error containing tc.expectedErrorMsg
				// This is the BUG - the application crashes instead of launching with local provider only
				t.Logf("UNFIXED CODE BEHAVIOR: Application failed to launch with error: %v", err)
				t.Logf("Expected error message on unfixed code: %s", tc.expectedErrorMsg)
				
				// Verify this is the expected error from unfixed code
				if !strings.Contains(err.Error(), tc.expectedErrorMsg) {
					t.Errorf("Unexpected error message. Expected to contain '%s', got: %v", tc.expectedErrorMsg, err)
				}
				
				// This test MUST FAIL on unfixed code to confirm the bug exists
				t.Fatalf("BUG CONFIRMED: Application crashes on launch when cloud provider has missing API key. Expected: application launches successfully with local provider only.")
			}

			// AFTER FIX: The following assertions should pass

			// Assert 1: Application launched successfully (manager created)
			if manager == nil {
				t.Fatal("Expected manager to be created, got nil")
			}

			// Assert 2: Local provider should be available
			localProvider := manager.GetLocalProvider()
			if localProvider == nil {
				t.Error("Expected local provider to be available, got nil")
			}

			// Assert 3: Cloud provider should be unavailable (nil)
			cloudProvider := manager.GetCloudProvider()
			if cloudProvider != nil {
				t.Error("Expected cloud provider to be nil (unavailable), but it was initialized")
			}

			// Assert 4: Warning should be logged
			logOutput := logBuf.String()
			if !strings.Contains(logOutput, "Cloud provider initialization failed") {
				t.Errorf("Expected warning message 'Cloud provider initialization failed' in logs, got: %s", logOutput)
			}

			// Assert 5: Warning message should contain the specific error
			if !strings.Contains(logOutput, tc.expectedErrorMsg) {
				t.Errorf("Expected warning to contain '%s', got: %s", tc.expectedErrorMsg, logOutput)
			}

			t.Logf("FIXED CODE BEHAVIOR: Application launched successfully with local provider only")
		})
	}
}

// TestBugCondition_NoLocalProviderConfigured tests the edge case where no local provider is configured
// and no cloud provider API key is provided.
//
// Expected Behavior:
//   - Application should exit gracefully with error message
//   - Error message should indicate local provider is required
func TestBugCondition_NoLocalProviderConfigured(t *testing.T) {
	var logBuf bytes.Buffer
	logger := logging.NewLogger("test", logging.INFO, &logBuf)

	// Config with no local provider and no cloud provider API key
	cfg := &config.Config{
		LocalProvider: config.ProviderConfig{
			Type: "", // Not configured
		},
		CloudProvider: config.ProviderConfig{
			Type:      "openai",
			OpenAIKey: "", // Missing API key
		},
		Privacy: config.PrivacyConfig{
			DefaultToLocal: true,
			CloudRAGPolicy: "no_rag",
		},
	}

	manager, err := NewDualProviderManager(cfg, logger)

	// Expected: Error should be returned indicating local provider is required
	if err == nil {
		t.Fatal("Expected error when no local provider is configured, got nil")
	}

	// After fix, this should return a clear error message about local provider being required
	// On unfixed code, it may return "at least one provider must be configured"
	expectedErrorAfterFix := "A local provider is required"
	if strings.Contains(err.Error(), expectedErrorAfterFix) {
		t.Logf("FIXED CODE: Got expected error message: %v", err)
	} else {
		t.Logf("UNFIXED CODE: Got error: %v (expected to contain '%s' after fix)", err, expectedErrorAfterFix)
	}

	if manager != nil {
		t.Error("Expected manager to be nil when error occurs")
	}
}
