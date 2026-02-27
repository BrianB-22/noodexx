# Cloud Provider Optional Launch Bugfix Design

## Overview

The application currently crashes on launch when a cloud provider (OpenAI/Anthropic) is configured in `config.json` but lacks an API key. This prevents users from running the application with only the local provider (Ollama), which should be fully functional independently. This bugfix implements graceful degradation to allow the application to launch successfully when cloud provider credentials are missing, while maintaining full functionality with the local provider.

The fix modifies the provider initialization logic in `dual_manager.go` to treat cloud provider initialization failures as warnings rather than fatal errors, validates that the local provider is properly configured (mandatory requirement), and updates the UI to disable cloud provider controls when the cloud provider is not available.

## Glossary

- **Bug_Condition (C)**: The condition that triggers the bug - when the application launches with a cloud provider configured but missing API credentials
- **Property (P)**: The desired behavior when the bug condition holds - the application should launch successfully with only the local provider enabled
- **Preservation**: Existing functionality when both providers are properly configured must remain unchanged
- **DualProviderManager**: The component in `noodexx/internal/provider/dual_manager.go` that manages both local and cloud provider instances
- **NewProvider**: The function in `noodexx/internal/llm/provider.go` that validates API keys and returns an error when they are missing
- **Local Provider**: The mandatory Ollama provider that runs locally and does not require API keys
- **Cloud Provider**: The optional OpenAI or Anthropic provider that requires API keys for operation
- **Graceful Degradation**: The system's ability to continue operating with reduced functionality when some components fail

## Bug Details

### Fault Condition

The bug manifests when the application launches with a cloud provider configured in `config.json` but the required API key is not provided. The `NewDualProviderManager` function in `dual_manager.go` calls `llm.NewProvider` for the cloud provider, which validates the API key and returns an error if it's missing. This error is propagated up to `main.go`, causing the application to exit before the local provider can be used.

**Formal Specification:**
```
FUNCTION isBugCondition(input)
  INPUT: input of type ApplicationLaunchConfig
  OUTPUT: boolean
  
  RETURN input.CloudProvider.Type IN ['openai', 'anthropic']
         AND (
           (input.CloudProvider.Type == 'openai' AND input.CloudProvider.OpenAIKey == "")
           OR
           (input.CloudProvider.Type == 'anthropic' AND input.CloudProvider.AnthropicKey == "")
         )
         AND input.LocalProvider.Type == 'ollama'
         AND input.LocalProvider.OllamaEndpoint != ""
END FUNCTION
```

### Examples

- **Example 1**: User has `config.json` with `CloudProvider.Type = "openai"` and `CloudProvider.OpenAIKey = ""`, but `LocalProvider.Type = "ollama"` is properly configured. Expected: Application launches with local provider only. Actual: Application crashes with "failed to initialize cloud provider: openai API key is required"

- **Example 2**: User has `config.json` with `CloudProvider.Type = "anthropic"` and `CloudProvider.AnthropicKey = ""`, but `LocalProvider.Type = "ollama"` is properly configured. Expected: Application launches with local provider only. Actual: Application crashes with "failed to initialize cloud provider: anthropic API key is required"

- **Example 3**: User removes cloud provider API key from environment variables but leaves cloud provider configured in `config.json`. Expected: Application launches with warning message about missing cloud provider. Actual: Application crashes and prevents access to local provider

- **Edge Case**: User has no local provider configured and no cloud provider API key. Expected: Application exits gracefully with message "A local provider is required. Please refer to documentation on configuration."

## Expected Behavior

### Preservation Requirements

**Unchanged Behaviors:**
- When both local and cloud providers are properly configured with valid credentials, the application must continue to initialize both providers successfully
- When the application is running with a properly configured cloud provider, switching between providers via the UI toggle must continue to work exactly as before
- When the application is running with a properly configured cloud provider, all cloud provider functionality (chat, embeddings, RAG) must continue to work exactly as before
- When the application is running with the local provider, all local provider functionality must continue to work exactly as before regardless of cloud provider status

**Scope:**
All inputs that do NOT involve missing cloud provider credentials should be completely unaffected by this fix. This includes:
- Application launches with valid cloud provider API keys
- Provider switching functionality when both providers are available
- All API endpoints and handlers when both providers are configured
- Configuration validation and saving functionality
- RAG policy enforcement and privacy controls

## Hypothesized Root Cause

Based on the bug description and code analysis, the root causes are:

1. **Strict Error Propagation in Provider Initialization**: The `NewDualProviderManager` function in `dual_manager.go` (lines 68-72) treats cloud provider initialization errors as fatal, returning the error immediately and preventing the manager from being created even when the local provider is properly configured.

2. **Mandatory API Key Validation**: The `NewProvider` function in `llm/provider.go` (lines 56-58 for OpenAI, 61-63 for Anthropic) validates that API keys are present and returns an error if they are missing, with no option for graceful degradation.

3. **No Local Provider Validation**: The current code does not validate that the local provider is properly configured before allowing the application to launch, which could lead to an application running with no functional providers.

4. **UI Does Not Reflect Provider Availability**: The chat interface toggle and settings page do not check whether the cloud provider is actually available before allowing users to interact with cloud provider controls.

## Correctness Properties

Property 1: Fault Condition - Application Launches with Missing Cloud Provider Credentials

_For any_ application launch configuration where a cloud provider is configured but API credentials are missing (isBugCondition returns true), the fixed initialization SHALL launch the application successfully with only the local provider enabled, log a warning message about the missing cloud provider configuration, and set the cloud provider instance to nil in the DualProviderManager.

**Validates: Requirements 2.1, 2.2, 2.3**

Property 2: Preservation - Dual Provider Functionality with Valid Credentials

_For any_ application launch configuration where both local and cloud providers have valid credentials (isBugCondition returns false), the fixed initialization SHALL produce exactly the same behavior as the original code, initializing both providers successfully and allowing full switching functionality between them.

**Validates: Requirements 3.1, 3.2, 3.3, 3.4**

## Fix Implementation

### Changes Required

Assuming our root cause analysis is correct:

**File**: `noodexx/internal/provider/dual_manager.go`

**Function**: `NewDualProviderManager`

**Specific Changes**:
1. **Graceful Cloud Provider Initialization**: Modify the cloud provider initialization block (lines 60-75) to log a warning and continue when `llm.NewProvider` returns an error, rather than returning the error immediately. Set `manager.cloudProvider = nil` when initialization fails.

2. **Mandatory Local Provider Validation**: After attempting to initialize both providers, add validation to ensure the local provider is configured. If `manager.localProvider == nil`, return an error with message "A local provider is required. Please refer to documentation on configuration."

3. **Update Error Message**: Change the final validation message from "at least one provider (local or cloud) must be configured" to reflect that the local provider is mandatory.

4. **Add Warning Logging**: When cloud provider initialization fails, log a clear warning message: "Cloud provider initialization failed: %v. Application will run with local provider only."

**File**: `noodexx/internal/provider/dual_manager.go`

**Function**: `Reload`

**Specific Changes**:
1. **Consistent Graceful Degradation**: Apply the same graceful degradation logic to the `Reload` method (lines 207-217) so that cloud provider initialization failures during configuration reload are handled the same way as during initial startup.

2. **Maintain Local Provider Requirement**: Ensure the reload validation also enforces that the local provider must be configured.

**File**: `noodexx/main.go`

**Function**: `main`

**Specific Changes**:
1. **Display Cloud Provider Status**: Update the cloud provider display section (lines 119-133) to check if the cloud provider was successfully initialized and display an appropriate message when it's not available.

2. **Conditional Active Provider Check**: Modify the active provider check (lines 165-168) to handle the case where `GetActiveProvider` returns an error because the cloud provider is not configured but the user's default is set to cloud mode. In this case, log a warning "Defaulting to Local AI because no cloud provider configured" and switch to local mode automatically by updating the configuration's default provider setting.

**File**: `noodexx/web/templates/chat.html`

**Function**: JavaScript initialization and provider switching

**Specific Changes**:
1. **Disable Cloud Provider Toggle**: Add logic to check if the cloud provider is available when the page loads. If not available, disable the cloud provider radio button and add a visual indicator (e.g., grayed out with tooltip).

2. **Add Data Attribute**: Add a data attribute to the template (e.g., `data-cloud-available="{{.CloudProviderAvailable}}"`) that the JavaScript can read to determine if the cloud provider is available.

**File**: `noodexx/web/templates/settings.html`

**Function**: Cloud provider configuration display

**Specific Changes**:
1. **Conditional Test Button**: Modify the cloud provider section (around line 155) to check if the cloud provider has valid credentials. If not, replace the "Test Cloud Connection" button with a message: "Please configure cloud provider in config.json file and restart service."

2. **Add Configuration Status Indicator**: Add a visual indicator (badge or icon) showing whether the cloud provider is properly configured and available.

**File**: `noodexx/internal/api/handlers.go` or `noodexx/internal/api/server.go`

**Function**: Template data preparation for chat and settings pages

**Specific Changes**:
1. **Add CloudProviderAvailable Field**: Add a boolean field to the template data that indicates whether the cloud provider is available (not nil in the DualProviderManager).

2. **Pass Provider Status to Templates**: Ensure this field is populated when rendering the chat and settings templates.

## Testing Strategy

### Validation Approach

The testing strategy follows a two-phase approach: first, surface counterexamples that demonstrate the bug on unfixed code, then verify the fix works correctly and preserves existing behavior.

### Exploratory Fault Condition Checking

**Goal**: Surface counterexamples that demonstrate the bug BEFORE implementing the fix. Confirm or refute the root cause analysis. If we refute, we will need to re-hypothesize.

**Test Plan**: Create test configurations with missing cloud provider credentials and attempt to launch the application. Run these tests on the UNFIXED code to observe failures and understand the root cause. Verify that the error originates from `llm.NewProvider` and propagates through `NewDualProviderManager` to `main.go`.

**Test Cases**:
1. **OpenAI Missing Key Test**: Configure `config.json` with `CloudProvider.Type = "openai"` and `CloudProvider.OpenAIKey = ""`, with valid local provider. Launch application. (will fail on unfixed code with "openai API key is required")

2. **Anthropic Missing Key Test**: Configure `config.json` with `CloudProvider.Type = "anthropic"` and `CloudProvider.AnthropicKey = ""`, with valid local provider. Launch application. (will fail on unfixed code with "anthropic API key is required")

3. **No Local Provider Test**: Configure `config.json` with no local provider and no cloud provider API key. Launch application. (may fail on unfixed code with "at least one provider must be configured" or succeed incorrectly)

4. **Environment Variable Override Test**: Configure cloud provider in `config.json` but remove API key from environment variables. Launch application. (will fail on unfixed code)

**Expected Counterexamples**:
- Application exits with error "Failed to initialize provider manager: failed to initialize cloud provider: [provider] API key is required"
- Possible causes: strict error propagation in `NewDualProviderManager`, mandatory API key validation in `llm.NewProvider`, no graceful degradation path

### Fix Checking

**Goal**: Verify that for all inputs where the bug condition holds, the fixed function produces the expected behavior.

**Pseudocode:**
```
FOR ALL config WHERE isBugCondition(config) DO
  result := LaunchApplication_fixed(config)
  ASSERT result.launched == true
  ASSERT result.localProviderAvailable == true
  ASSERT result.cloudProviderAvailable == false
  ASSERT result.warningLogged == true
  ASSERT result.warningMessage CONTAINS "Cloud provider initialization failed"
END FOR
```

### Preservation Checking

**Goal**: Verify that for all inputs where the bug condition does NOT hold, the fixed function produces the same result as the original function.

**Pseudocode:**
```
FOR ALL config WHERE NOT isBugCondition(config) DO
  ASSERT LaunchApplication_original(config) = LaunchApplication_fixed(config)
END FOR
```

**Testing Approach**: Property-based testing is recommended for preservation checking because:
- It generates many test cases automatically across the input domain
- It catches edge cases that manual unit tests might miss
- It provides strong guarantees that behavior is unchanged for all non-buggy inputs

**Test Plan**: Observe behavior on UNFIXED code first for valid configurations with both providers, then write property-based tests capturing that behavior.

**Test Cases**:
1. **Both Providers Valid Preservation**: Configure both local and cloud providers with valid credentials. Verify application launches successfully, both providers are available, and provider switching works correctly. (should work identically on both unfixed and fixed code)

2. **Provider Switching Preservation**: With both providers configured, verify that switching between providers via the UI toggle continues to work exactly as before. (should work identically on both unfixed and fixed code)

3. **Cloud Provider Functionality Preservation**: With valid cloud provider credentials, verify that all cloud provider operations (chat, embeddings, RAG) continue to work exactly as before. (should work identically on both unfixed and fixed code)

4. **Configuration Reload Preservation**: With valid credentials, verify that configuration reload continues to work correctly. (should work identically on both unfixed and fixed code)

### Unit Tests

- Test `NewDualProviderManager` with missing cloud provider API key (should succeed with warning)
- Test `NewDualProviderManager` with missing local provider (should fail with clear error message)
- Test `NewDualProviderManager` with both providers valid (should succeed with both providers initialized)
- Test `Reload` method with cloud provider credentials removed (should succeed with warning)
- Test `GetActiveProvider` when cloud provider is nil and default is cloud mode (should return error or fallback to local)
- Test UI template rendering with cloud provider unavailable (should disable toggle and show message)

### Property-Based Tests

- Generate random configurations with various combinations of provider settings and verify that the application always launches when local provider is configured
- Generate random configurations with valid credentials and verify that all functionality is preserved
- Generate random sequences of provider switches and verify that the application handles them correctly regardless of cloud provider availability

### Integration Tests

- Test full application launch flow with missing cloud provider credentials
- Test that chat functionality works correctly with only local provider available
- Test that settings page displays correct status for unconfigured cloud provider
- Test that provider toggle is disabled in UI when cloud provider is not available
- Test that configuration reload handles cloud provider credential changes correctly
- Test that warning messages are displayed to users when cloud provider is not configured
