# Testing Guide

## Purpose

This document provides comprehensive guidance on testing strategies, patterns, and best practices for Noodexx development.

---

## Testing Philosophy

### Bug Condition Testing
**Goal**: Prove the bug exists, then prove it's fixed

1. Write test that encodes expected behavior
2. Run on unfixed code (should FAIL)
3. Document counterexamples
4. Implement fix
5. Run same test (should PASS)

### Preservation Testing
**Goal**: Ensure existing functionality is not broken

1. Observe behavior on unfixed code
2. Write tests capturing that behavior
3. Run on unfixed code (should PASS)
4. Implement fix
5. Run same tests (should still PASS)

### Property-Based Testing
**Goal**: Generate many test cases for stronger guarantees

- Test properties that should hold for all inputs
- Framework generates random test cases
- Catches edge cases manual tests might miss
- Provides confidence in correctness

---

## Test Organization

### Test File Naming
- Unit tests: `{package}_test.go` in same directory
- Integration tests: `{feature}_integration_test.go`
- Component tests: `{component}_test.go`

### Test Function Naming
```go
// Unit test
func TestFunctionName_Scenario(t *testing.T)

// Bug condition test
func TestBugCondition_DescriptionOfBug(t *testing.T)

// Preservation test
func TestPreservation_FeatureThatMustWork(t *testing.T)

// Integration test
func TestIntegration_EndToEndScenario(t *testing.T)
```

---

## Unit Testing Patterns

### Basic Unit Test
```go
func TestChunker_ChunkText(t *testing.T) {
    chunker := rag.NewChunker(500, 50)
    
    text := "This is a test document with multiple sentences. " +
            "It should be split into chunks based on token count."
    
    chunks := chunker.ChunkText(text)
    
    // Assertions
    if len(chunks) == 0 {
        t.Error("Expected at least one chunk")
    }
    
    for i, chunk := range chunks {
        if len(chunk) == 0 {
            t.Errorf("Chunk %d is empty", i)
        }
    }
}
```

### Table-Driven Tests
```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "user@example.com", false},
        {"missing @", "userexample.com", true},
        {"missing domain", "user@", true},
        {"empty", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Testing with Mocks
```go
type mockStore struct {
    chunks []Chunk
    err    error
}

func (m *mockStore) SaveChunk(ctx context.Context, userID int64, source, text string, embedding []float32, tags []string, summary string) error {
    if m.err != nil {
        return m.err
    }
    m.chunks = append(m.chunks, Chunk{
        UserID: userID,
        Source: source,
        Text:   text,
    })
    return nil
}

func TestIngester_IngestText(t *testing.T) {
    mockStore := &mockStore{}
    mockProvider := &mockProvider{}
    
    ingester := ingest.NewIngester(mockStore, mockProvider, logger)
    
    err := ingester.IngestText(context.Background(), 1, "test.txt", "content", []string{"tag"})
    
    if err != nil {
        t.Fatalf("IngestText failed: %v", err)
    }
    
    if len(mockStore.chunks) == 0 {
        t.Error("Expected chunks to be saved")
    }
}
```

---

## Integration Testing Patterns

### HTTP Handler Integration Test
```go
func TestHandleAsk_Integration(t *testing.T) {
    // Setup: Create real dependencies (or test doubles)
    db, err := store.NewStore(":memory:", "single")
    if err != nil {
        t.Fatalf("Failed to create store: %v", err)
    }
    defer db.Close()
    
    // Create test user
    user, _ := db.CreateUser(context.Background(), "testuser", "password", "test@example.com", false)
    
    // Create server with real dependencies
    server := &api.Server{
        store:           db,
        provider:        &mockProvider{},
        logger:          logging.NewLogger("test", logging.INFO, io.Discard),
        providerManager: &mockProviderManager{},
    }
    
    // Create request
    reqBody := `{"query":"What is RAG?","session_id":"test-session"}`
    req := httptest.NewRequest("POST", "/api/ask", strings.NewReader(reqBody))
    req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, user.ID))
    
    // Execute request
    w := httptest.NewRecorder()
    server.handleAsk(w, req)
    
    // Assert response
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
    
    // Verify message was saved
    messages, _ := db.GetChatMessages(context.Background(), user.ID, "test-session")
    if len(messages) == 0 {
        t.Error("Expected message to be saved")
    }
}
```

### Database Integration Test
```go
func TestStore_SaveAndSearchChunks_Integration(t *testing.T) {
    // Create in-memory database
    db, err := store.NewStore(":memory:", "single")
    if err != nil {
        t.Fatalf("Failed to create store: %v", err)
    }
    defer db.Close()
    
    ctx := context.Background()
    
    // Create test user
    user, _ := db.CreateUser(ctx, "testuser", "password", "test@example.com", false)
    
    // Save chunks
    embedding1 := make([]float32, 384)
    for i := range embedding1 {
        embedding1[i] = 0.1
    }
    
    err = db.SaveChunk(ctx, user.ID, "doc1.txt", "This is chunk 1", embedding1, []string{"tag1"}, "Summary 1")
    if err != nil {
        t.Fatalf("Failed to save chunk: %v", err)
    }
    
    // Search chunks
    queryEmbedding := make([]float32, 384)
    for i := range queryEmbedding {
        queryEmbedding[i] = 0.1
    }
    
    results, err := db.SearchChunks(ctx, user.ID, queryEmbedding, 10)
    if err != nil {
        t.Fatalf("Failed to search chunks: %v", err)
    }
    
    if len(results) == 0 {
        t.Error("Expected to find chunks")
    }
}
```

---

## Bug Condition Testing Pattern

### Example from cloud-provider-optional-launch bugfix

```go
func TestBugCondition_ApplicationLaunchWithMissingCloudProviderCredentials(t *testing.T) {
    testCases := []struct {
        name                string
        cloudProviderType   string
        cloudProviderAPIKey string
        expectedErrorMsg    string // Expected error on UNFIXED code
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
            }
            
            // Attempt to create DualProviderManager
            manager, err := provider.NewDualProviderManager(cfg, logger)
            
            // EXPECTED BEHAVIOR (after fix):
            // - No error should be returned
            // - Local provider should be available
            // - Cloud provider should be nil
            // - Warning should be logged
            
            if err != nil {
                // ON UNFIXED CODE: This will fail
                t.Logf("UNFIXED CODE: Application failed to launch with error: %v", err)
                t.Fatalf("BUG CONFIRMED: Application crashes instead of launching with local provider only")
            }
            
            // AFTER FIX: These assertions should pass
            
            if manager == nil {
                t.Fatal("Expected manager to be created, got nil")
            }
            
            if manager.GetLocalProvider() == nil {
                t.Error("Expected local provider to be available")
            }
            
            if manager.GetCloudProvider() != nil {
                t.Error("Expected cloud provider to be nil")
            }
            
            logOutput := logBuf.String()
            if !strings.Contains(logOutput, "Cloud provider initialization failed") {
                t.Errorf("Expected warning in logs, got: %s", logOutput)
            }
        })
    }
}
```

---

## Preservation Testing Pattern

```go
func TestPreservation_DualProviderFunctionalityWithValidCredentials(t *testing.T) {
    testCases := []struct {
        name              string
        cloudProviderType string
        cloudProviderKey  string
        defaultToLocal    bool
    }{
        {
            name:              "OpenAI with valid key - default to local",
            cloudProviderType: "openai",
            cloudProviderKey:  "sk-test-key-12345678901234567890123456789012",
            defaultToLocal:    true,
        },
        {
            name:              "OpenAI with valid key - default to cloud",
            cloudProviderType: "openai",
            cloudProviderKey:  "sk-test-key-12345678901234567890123456789012",
            defaultToLocal:    false,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            var logBuf bytes.Buffer
            logger := logging.NewLogger("test", logging.INFO, &logBuf)
            
            // Create config with BOTH providers having valid credentials
            cfg := &config.Config{
                LocalProvider: config.ProviderConfig{
                    Type:             "ollama",
                    OllamaEndpoint:   "http://localhost:11434",
                    OllamaEmbedModel: "nomic-embed-text",
                    OllamaChatModel:  "llama3.2",
                },
                CloudProvider: config.ProviderConfig{
                    Type:      tc.cloudProviderType,
                    OpenAIKey: tc.cloudProviderKey,
                },
                Privacy: config.PrivacyConfig{
                    DefaultToLocal: tc.defaultToLocal,
                },
            }
            
            manager, err := provider.NewDualProviderManager(cfg, logger)
            
            // PRESERVATION: Both providers should initialize successfully
            if err != nil {
                t.Fatalf("Expected successful initialization, got error: %v", err)
            }
            
            if manager.GetLocalProvider() == nil {
                t.Error("Expected local provider to be initialized")
            }
            
            if manager.GetCloudProvider() == nil {
                t.Error("Expected cloud provider to be initialized")
            }
            
            // PRESERVATION: Provider switching should work
            if manager.IsLocalMode() != tc.defaultToLocal {
                t.Errorf("Expected IsLocalMode()=%v, got %v", tc.defaultToLocal, manager.IsLocalMode())
            }
            
            activeProvider, err := manager.GetActiveProvider()
            if err != nil {
                t.Errorf("Expected GetActiveProvider to succeed, got error: %v", err)
            }
            
            if activeProvider == nil {
                t.Error("Expected active provider to be non-nil")
            }
        })
    }
}
```

---

## Testing Best Practices

### 1. Test Isolation
- Each test should be independent
- Use setup/teardown for shared resources
- Don't rely on test execution order

```go
func TestMyFeature(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    defer db.Close()
    
    // Test
    // ...
    
    // Cleanup happens automatically via defer
}
```

### 2. Use Subtests for Related Cases
```go
func TestUserValidation(t *testing.T) {
    t.Run("valid user", func(t *testing.T) {
        // Test valid case
    })
    
    t.Run("invalid email", func(t *testing.T) {
        // Test invalid email
    })
    
    t.Run("missing password", func(t *testing.T) {
        // Test missing password
    })
}
```

### 3. Test Error Cases
```go
func TestDivide(t *testing.T) {
    t.Run("normal division", func(t *testing.T) {
        result, err := Divide(10, 2)
        if err != nil {
            t.Fatalf("Unexpected error: %v", err)
        }
        if result != 5 {
            t.Errorf("Expected 5, got %f", result)
        }
    })
    
    t.Run("division by zero", func(t *testing.T) {
        _, err := Divide(10, 0)
        if err == nil {
            t.Error("Expected error for division by zero")
        }
    })
}
```

### 4. Use Test Helpers
```go
func setupTestServer(t *testing.T) *api.Server {
    t.Helper() // Marks this as a helper function
    
    db, err := store.NewStore(":memory:", "single")
    if err != nil {
        t.Fatalf("Failed to create store: %v", err)
    }
    
    return &api.Server{
        store:  db,
        logger: logging.NewLogger("test", logging.INFO, io.Discard),
    }
}

func TestMyHandler(t *testing.T) {
    server := setupTestServer(t)
    // Test using server
}
```

### 5. Test Context Cancellation
```go
func TestLongRunningOperation_Cancellation(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    err := LongRunningOperation(ctx)
    
    if err != context.DeadlineExceeded {
        t.Errorf("Expected context.DeadlineExceeded, got %v", err)
    }
}
```

---

## Running Tests

### Run All Tests
```bash
go test ./...
```

### Run Tests in Specific Package
```bash
go test ./internal/api
```

### Run Specific Test
```bash
go test -run TestBugCondition_ApplicationLaunch ./internal/provider
```

### Run with Verbose Output
```bash
go test -v ./...
```

### Run with Coverage
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run with Race Detector
```bash
go test -race ./...
```

---

## Mock Generation

### Manual Mocks
```go
type mockStore struct {
    saveChunkFunc func(ctx context.Context, userID int64, source, text string, embedding []float32, tags []string, summary string) error
}

func (m *mockStore) SaveChunk(ctx context.Context, userID int64, source, text string, embedding []float32, tags []string, summary string) error {
    if m.saveChunkFunc != nil {
        return m.saveChunkFunc(ctx, userID, source, text, embedding, tags, summary)
    }
    return nil
}
```

### Using Mocks in Tests
```go
func TestWithCustomMock(t *testing.T) {
    mock := &mockStore{
        saveChunkFunc: func(ctx context.Context, userID int64, source, text string, embedding []float32, tags []string, summary string) error {
            // Custom behavior for this test
            if source == "error.txt" {
                return fmt.Errorf("simulated error")
            }
            return nil
        },
    }
    
    // Use mock in test
}
```

---

## Common Testing Pitfalls

### 1. Testing Implementation Instead of Behavior
❌ Bad:
```go
func TestUserService_CallsDatabase(t *testing.T) {
    // Testing that a specific method is called
}
```

✅ Good:
```go
func TestUserService_CreatesUser(t *testing.T) {
    // Testing that the user is created (behavior)
}
```

### 2. Brittle Tests
❌ Bad:
```go
if response != "User created with ID 123 at 2024-01-15" {
    t.Error("Unexpected response")
}
```

✅ Good:
```go
if !strings.Contains(response, "User created") {
    t.Error("Expected success message")
}
```

### 3. Not Testing Edge Cases
✅ Always test:
- Empty inputs
- Nil values
- Boundary conditions
- Error cases
- Concurrent access (if applicable)

---

## Next Steps

For more guidance, see:
- `05-ui-development.md` - Frontend testing and development
- `06-troubleshooting.md` - Common issues and solutions
