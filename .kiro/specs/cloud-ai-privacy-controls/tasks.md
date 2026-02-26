# Implementation Plan: Cloud AI Privacy Controls

## Overview

This implementation plan transforms Noodexx from a single-provider architecture to a dual-provider system with granular privacy controls. The work is organized into discrete coding tasks that build incrementally, starting with core configuration changes, then implementing provider management, RAG policy enforcement, UI updates, and finally testing and validation.

## Tasks

- [x] 1. Update configuration data structures for dual providers
  - Modify `internal/config/config.go` to add `LocalProvider` and `CloudProvider` fields
  - Update `ProviderConfig` struct to support all provider types (Ollama, OpenAI, Anthropic)
  - Update `PrivacyConfig` struct to add `UseLocalAI` bool and `CloudRAGPolicy` string fields
  - Add JSON tags for proper serialization
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 3.1, 3.2, 3.3, 3.4_

- [ ]* 1.1 Write property test for configuration round-trip preservation
  - **Property 1: Configuration Round-Trip Preservation**
  - **Validates: Requirements 1.3, 1.4, 2.4, 2.5, 3.3, 3.4, 9.1, 9.2, 9.3, 9.4**
  - Test that any valid dual-provider configuration can be saved and loaded without data loss

- [ ] 2. Implement configuration validation and migration
  - [x] 2.1 Add validation methods to ProviderConfig
    - Implement `ValidateLocal()` method to validate Ollama configuration
    - Implement `ValidateCloud()` method to validate OpenAI/Anthropic configuration
    - Implement `ValidateRAGPolicy()` method to validate RAG policy values
    - _Requirements: 8.1, 8.2, 8.3, 10.5_
  
  - [ ]* 2.2 Write property test for configuration validation errors
    - **Property 8: Configuration Validation Errors**
    - **Validates: Requirements 10.5**
    - Test that invalid configurations produce appropriate validation errors
  
  - [x] 2.3 Implement backward compatibility migration
    - Add migration logic in config loading to convert old single-provider format to dual-provider format
    - Detect old config format (missing LocalProvider/CloudProvider fields)
    - Migrate old provider config to appropriate new field based on provider type
    - Set safe defaults for new fields (UseLocalAI based on old privacy mode, CloudRAGPolicy to "no_rag")
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_
  
  - [ ]* 2.4 Write unit tests for backward compatibility migration
    - Test migration from Ollama-only config
    - Test migration from OpenAI-only config
    - Test migration from Anthropic-only config
    - Test that new configs are not migrated
    - _Requirements: 9.1, 9.2, 9.3, 9.4_

- [ ] 3. Create DualProviderManager component
  - [x] 3.1 Create DualProviderManager struct and constructor
    - Create new file `internal/provider/dual_manager.go`
    - Define `DualProviderManager` struct with localProvider, cloudProvider, config, and logger fields
    - Implement `NewDualProviderManager()` constructor that initializes both providers
    - Handle cases where providers are not configured (Type is empty string)
    - _Requirements: 1.5, 2.1, 2.2_
  
  - [x] 3.2 Implement provider routing methods
    - Implement `GetActiveProvider()` method that returns provider based on Privacy.UseLocalAI
    - Implement `IsLocalMode()` method that returns current toggle state
    - Implement `GetProviderName()` method that returns human-readable provider name for UI
    - Return appropriate errors when active provider is not configured
    - _Requirements: 2.1, 2.2, 7.1, 7.2, 8.1, 8.2, 8.3_
  
  - [ ]* 3.3 Write property test for provider routing correctness
    - **Property 2: Provider Routing Correctness**
    - **Validates: Requirements 2.1, 2.2**
    - Test that queries route to correct provider based on toggle state
  
  - [ ]* 3.4 Write property test for dual provider coexistence
    - **Property 3: Dual Provider Coexistence**
    - **Validates: Requirements 1.5**
    - Test that both providers can be configured simultaneously without interference
  
  - [x] 3.5 Implement configuration reload method
    - Implement `Reload()` method to reinitialize providers after config changes
    - Handle provider initialization errors gracefully
    - _Requirements: 2.3, 10.4_
  
  - [ ]* 3.6 Write unit tests for DualProviderManager
    - Test GetActiveProvider with local mode enabled
    - Test GetActiveProvider with cloud mode enabled
    - Test error handling for unconfigured local provider
    - Test error handling for unconfigured cloud provider
    - Test GetProviderName returns correct names
    - _Requirements: 2.1, 2.2, 7.1, 7.2, 8.1, 8.2, 8.3_

- [ ] 4. Create RAGPolicyEnforcer component
  - [x] 4.1 Create RAGPolicyEnforcer struct and methods
    - Create new file `internal/rag/policy_enforcer.go`
    - Define `RAGPolicyEnforcer` struct with config and logger fields
    - Implement `NewRAGPolicyEnforcer()` constructor
    - Implement `ShouldPerformRAG()` method with decision logic (always true for local, policy-based for cloud)
    - Implement `GetRAGStatus()` method that returns human-readable status string
    - _Requirements: 4.1, 4.2, 4.3, 5.1, 5.2, 5.3, 6.1, 6.2, 6.3_
  
  - [ ]* 4.2 Write property test for local mode RAG invariance
    - **Property 4: Local Mode RAG Invariance**
    - **Validates: Requirements 4.1, 4.2, 4.3**
    - Test that local mode always performs RAG regardless of policy setting
  
  - [ ]* 4.3 Write property test for cloud mode no-RAG enforcement
    - **Property 5: Cloud Mode No-RAG Enforcement**
    - **Validates: Requirements 5.1, 5.2**
    - Test that cloud mode with "no_rag" policy disables RAG
  
  - [ ]* 4.4 Write property test for cloud mode allow-RAG enablement
    - **Property 6: Cloud Mode Allow-RAG Enablement**
    - **Validates: Requirements 6.1, 6.2**
    - Test that cloud mode with "allow_rag" policy enables RAG
  
  - [ ]* 4.5 Write unit tests for RAGPolicyEnforcer
    - Test ShouldPerformRAG with all combinations of toggle state and policy
    - Test GetRAGStatus returns correct status strings
    - _Requirements: 4.1, 4.2, 4.3, 5.1, 5.2, 5.3, 6.1, 6.2, 6.3_

- [x] 5. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 6. Modify query handler to use dual providers and enforce RAG policy
  - [x] 6.1 Update Server struct to include new components
    - Add `providerManager *DualProviderManager` field to Server struct in `internal/api/handlers.go`
    - Add `ragEnforcer *RAGPolicyEnforcer` field to Server struct
    - Update server initialization to create these components
    - _Requirements: 2.1, 2.2, 4.1, 5.1, 6.1_
  
  - [x] 6.2 Modify handleAsk to use active provider
    - Replace direct provider usage with `providerManager.GetActiveProvider()`
    - Handle errors when active provider is not configured
    - Return HTTP 400 with appropriate error message for unconfigured provider
    - _Requirements: 2.1, 2.2, 8.1, 8.2, 8.3_
  
  - [x] 6.3 Modify handleAsk to conditionally perform RAG
    - Check `ragEnforcer.ShouldPerformRAG()` before performing RAG search
    - Skip embedding and search when RAG is disabled
    - Build prompt with or without chunks based on RAG decision
    - _Requirements: 4.1, 4.2, 5.1, 5.2, 6.1, 6.2_
  
  - [x] 6.4 Add provider and RAG status to response headers
    - Add `X-Provider-Name` header with provider name from `providerManager.GetProviderName()`
    - Add `X-RAG-Status` header with status from `ragEnforcer.GetRAGStatus()`
    - _Requirements: 7.1, 7.2, 7.3, 5.3, 6.3_
  
  - [x] 6.5 Write integration tests for modified handleAsk
    - Test query with local provider and RAG enabled
    - Test query with cloud provider and no-RAG policy
    - Test query with cloud provider and allow-RAG policy
    - Test error response for unconfigured local provider
    - Test error response for unconfigured cloud provider
    - _Requirements: 2.1, 2.2, 4.1, 4.2, 5.1, 5.2, 6.1, 6.2, 8.1, 8.2, 8.3_

- [ ]* 7. Write property test for unconfigured provider rejection
  - **Property 7: Unconfigured Provider Rejection**
  - **Validates: Requirements 8.1, 8.2, 8.3**
  - Test that queries fail appropriately when active provider is not configured

- [ ] 8. Update settings handler for dual provider configuration
  - [x] 8.1 Modify handleConfig to parse dual provider forms
    - Update form parsing to extract local_provider_* fields
    - Update form parsing to extract cloud_provider_* fields
    - Update form parsing to extract privacy toggle state (use_local_ai)
    - Update form parsing to extract RAG policy (cloud_rag_policy)
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 3.1, 3.2, 3.3, 3.4_
  
  - [x] 8.2 Add validation before saving configuration
    - Call ValidateLocal() on local provider config
    - Call ValidateCloud() on cloud provider config
    - Call ValidateRAGPolicy() on privacy config
    - Return HTTP 400 with field-specific error messages on validation failure
    - _Requirements: 8.1, 8.2, 8.3, 10.5_
  
  - [x] 8.3 Update config save and reload logic
    - Save updated config to disk
    - Call providerManager.Reload() to reinitialize providers
    - Return success confirmation within 1 second
    - _Requirements: 1.3, 1.4, 2.3, 2.4, 3.3, 3.4, 10.4_
  
  - [ ]* 8.4 Write unit tests for updated handleConfig
    - Test parsing and saving local provider config
    - Test parsing and saving cloud provider config
    - Test parsing and saving privacy toggle state
    - Test parsing and saving RAG policy
    - Test validation error responses
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 3.1, 3.2, 3.3, 3.4, 10.5_

- [ ] 9. Create privacy toggle API endpoint
  - [x] 9.1 Implement handlePrivacyToggle endpoint
    - Create new endpoint POST `/api/privacy-toggle` in `internal/api/handlers.go`
    - Parse toggle state from request body (local or cloud)
    - Update config.Privacy.UseLocalAI based on toggle state
    - Save config to disk
    - Return success response with new provider name and RAG status
    - Complete within 1 second
    - _Requirements: 2.3, 2.4, 7.3_
  
  - [ ]* 9.2 Write unit tests for handlePrivacyToggle
    - Test toggling to local mode
    - Test toggling to cloud mode
    - Test response includes correct provider name
    - Test response includes correct RAG status
    - Test config persistence
    - _Requirements: 2.3, 2.4, 7.3_

- [x] 10. Checkpoint - Ensure all backend tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 11. Update settings page UI for dual providers
  - [x] 11.1 Modify settings.html template structure
    - Add "Privacy Controls" section at top with default provider selection and RAG policy dropdown
    - Split provider configuration into "Local AI Provider" and "Cloud AI Provider" sections
    - Update form field names to use local_* and cloud_* prefixes
    - Add help text explaining local vs cloud provider modes
    - _Requirements: 1.1, 1.2, 3.1, 3.2, 10.1, 10.2, 10.3_
  
  - [x] 11.2 Add RAG policy dropdown with admin-only visibility
    - Add dropdown with "No RAG with Cloud AI" and "Allow RAG with Cloud AI" options
    - Add conditional rendering based on user admin status
    - Add explanatory text describing privacy implications of each option
    - _Requirements: 3.1, 3.2, 3.5_
  
  - [x] 11.3 Update settings form submission
    - Update JavaScript to collect all local provider fields
    - Update JavaScript to collect all cloud provider fields
    - Update JavaScript to collect privacy toggle state
    - Update JavaScript to collect RAG policy (if admin)
    - Display confirmation message on successful save
    - Display field-specific error messages on validation failure
    - _Requirements: 1.3, 1.4, 3.3, 3.4, 10.4, 10.5_
  
  - [ ]* 11.4 Write unit tests for settings page rendering
    - Test that both provider sections are displayed
    - Test that RAG policy dropdown is visible to admin
    - Test that RAG policy dropdown is hidden from non-admin
    - Test that help text is displayed
    - _Requirements: 1.1, 1.2, 3.1, 3.2, 10.1, 10.2, 10.3_

- [ ] 12. Add privacy toggle to chat UI
  - [x] 12.1 Create privacy toggle component in sidebar
    - Add toggle switch HTML to chat template (likely in a shared template or chat.html)
    - Add radio buttons for "Local AI" and "Cloud AI" options
    - Add icons (🔒 for local, ☁️ for cloud)
    - Style toggle to be prominent and easy to use
    - _Requirements: 2.3_
  
  - [x] 12.2 Implement toggle JavaScript functionality
    - Add `switchProvider()` JavaScript function
    - Send POST request to `/api/privacy-toggle` endpoint
    - Update UI immediately on successful toggle
    - Handle errors and display error messages
    - Disable toggle if selected provider is not configured
    - _Requirements: 2.3, 7.3, 8.1, 8.2_
  
  - [ ]* 12.3 Write unit tests for privacy toggle component
    - Test toggle switches between local and cloud
    - Test toggle sends correct API request
    - Test toggle updates UI on success
    - Test toggle displays error for unconfigured provider
    - _Requirements: 2.3, 7.3, 8.1, 8.2_

- [ ] 13. Add active provider and RAG status indicators to chat UI
  - [x] 13.1 Create status indicator component
    - Add status line to chat interface showing active provider name
    - Add RAG status display (enabled/disabled)
    - Style with color coding (green for local, blue for cloud)
    - Position prominently near chat session title
    - _Requirements: 7.1, 7.2, 5.3, 6.3_
  
  - [x] 13.2 Implement status update JavaScript
    - Read `X-Provider-Name` and `X-RAG-Status` headers from query responses
    - Update status indicator when headers are received
    - Update status indicator when privacy toggle changes
    - Ensure updates happen within 500ms of toggle change
    - _Requirements: 7.1, 7.2, 7.3, 5.3, 6.3_
  
  - [ ]* 13.3 Write unit tests for status indicators
    - Test indicator displays correct provider name
    - Test indicator displays correct RAG status
    - Test indicator updates on toggle change
    - Test indicator updates within 500ms
    - _Requirements: 7.1, 7.2, 7.3, 5.3, 6.3_

- [ ] 14. Add logout button to dashboard (multi-user mode)
  - [x] 14.1 Add logout button to dashboard template
    - Add logout button to dashboard.html or relevant template
    - Position button prominently (e.g., top-right corner)
    - Style consistently with existing UI
    - Only display in multi-user mode
    - _Requirements: Not explicitly in requirements, but mentioned in user context_
  
  - [x] 14.2 Implement logout functionality
    - Create POST `/api/logout` endpoint if not exists
    - Clear session token on logout
    - Redirect to login page
    - _Requirements: Not explicitly in requirements, but mentioned in user context_

- [ ] 15. Final checkpoint - Integration testing and validation
  - [x] 15.1 Test complete provider switch flow
    - Configure both local and cloud providers
    - Toggle between providers multiple times
    - Verify queries route to correct provider
    - Verify status indicators update correctly
    - _Requirements: 2.1, 2.2, 2.3, 7.1, 7.2, 7.3_
  
  - [x] 15.2 Test RAG policy enforcement flow
    - Set RAG policy to "No RAG with Cloud AI"
    - Submit queries in cloud mode and verify RAG is disabled
    - Set RAG policy to "Allow RAG with Cloud AI"
    - Submit queries in cloud mode and verify RAG is enabled
    - Submit queries in local mode and verify RAG is always enabled
    - _Requirements: 4.1, 4.2, 4.3, 5.1, 5.2, 5.3, 6.1, 6.2, 6.3_
  
  - [x] 15.3 Test settings persistence flow
    - Configure all settings
    - Restart application (or reload config)
    - Verify all settings are restored correctly
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_
  
  - [x] 15.4 Test error handling and recovery
    - Test unconfigured local provider error
    - Test unconfigured cloud provider error
    - Test invalid configuration validation errors
    - Verify error messages are clear and actionable
    - _Requirements: 8.1, 8.2, 8.3, 10.5_

- [x] 16. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation at key milestones
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples, edge cases, and UI components
- The implementation uses Go as specified in the design document
- All property-based tests use the gopter library with minimum 100 iterations
- Integration tests verify end-to-end workflows across multiple components
