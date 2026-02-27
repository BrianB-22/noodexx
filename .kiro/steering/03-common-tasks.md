# Common Development Tasks

## Purpose

This document provides step-by-step guidance for implementing common features and modifications in Noodexx.

---

## Adding a New API Endpoint

### 1. Define the Handler Function

**Location**: `internal/api/handlers.go` or create a new file for related handlers

```go
func (s *Server) handleMyNewEndpoint(w http.ResponseWriter, r *http.Request) {
    // Extract user_id from context (automatically injected by auth middleware)
    userID, err := auth.GetUserID(r.Context())
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Parse request body
    var req struct {
        Field1 string `json:"field1"`
        Field2 int    `json:"field2"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Perform operation using dependencies
    result, err := s.store.SomeOperation(r.Context(), userID, req.Field1)
    if err != nil {
        s.logger.Error("Operation failed: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    // Return JSON response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "data":    result,
    })
}
```

### 2. Register the Route

**Location**: `internal/api/server.go` in `RegisterRoutes()` method

```go
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
    // ... existing routes
    
    // Add your new route
    mux.HandleFunc("/api/my-endpoint", s.handleMyNewEndpoint)
}
```

### 3. Add Tests

**Location**: `internal/api/my_endpoint_test.go`

```go
func TestHandleMyNewEndpoint(t *testing.T) {
    // Create mock dependencies
    mockStore := &mockStore{}
    
    // Create test server
    server := &Server{
        store:  mockStore,
        logger: logging.NewLogger("test", logging.INFO, io.Discard),
    }
    
    // Create test request
    reqBody := `{"field1":"value","field2":42}`
    req := httptest.NewRequest("POST", "/api/my-endpoint", strings.NewReader(reqBody))
    req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, int64(1)))
    
    // Create response recorder
    w := httptest.NewRecorder()
    
    // Call handler
    server.handleMyNewEndpoint(w, req)
    
    // Assert response
    assert.Equal(t, http.StatusOK, w.Code)
    // ... more assertions
}
```

---

## Adding a New Database Table

### 1. Define the Model

**Location**: `internal/store/models.go`

```go
type MyNewModel struct {
    ID        int64     `json:"id"`
    UserID    int64     `json:"user_id"`
    Field1    string    `json:"field1"`
    Field2    int       `json:"field2"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 2. Create Migration

**Location**: `internal/store/migrations.go` - Add to `migrations` slice

```go
{
    Version: 15, // Increment from last version
    Description: "Add my_new_table",
    Up: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            CREATE TABLE IF NOT EXISTS my_new_table (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                user_id INTEGER NOT NULL,
                field1 TEXT NOT NULL,
                field2 INTEGER NOT NULL,
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
            )
        `)
        if err != nil {
            return fmt.Errorf("failed to create my_new_table: %w", err)
        }
        
        // Create index for user_id
        _, err = tx.ExecContext(ctx, `
            CREATE INDEX IF NOT EXISTS idx_my_new_table_user_id 
            ON my_new_table(user_id)
        `)
        return err
    },
},
```

### 3. Add CRUD Operations

**Location**: `internal/store/store.go` or create `internal/store/my_new_table.go`

```go
func (s *Store) CreateMyNewModel(ctx context.Context, userID int64, field1 string, field2 int) (*MyNewModel, error) {
    result, err := s.db.ExecContext(ctx, `
        INSERT INTO my_new_table (user_id, field1, field2)
        VALUES (?, ?, ?)
    `, userID, field1, field2)
    if err != nil {
        return nil, fmt.Errorf("failed to insert: %w", err)
    }
    
    id, _ := result.LastInsertId()
    return &MyNewModel{
        ID:        id,
        UserID:    userID,
        Field1:    field1,
        Field2:    field2,
        CreatedAt: time.Now(),
    }, nil
}

func (s *Store) GetMyNewModelsByUser(ctx context.Context, userID int64) ([]MyNewModel, error) {
    rows, err := s.db.QueryContext(ctx, `
        SELECT id, user_id, field1, field2, created_at
        FROM my_new_table
        WHERE user_id = ?
        ORDER BY created_at DESC
    `, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to query: %w", err)
    }
    defer rows.Close()
    
    var models []MyNewModel
    for rows.Next() {
        var m MyNewModel
        if err := rows.Scan(&m.ID, &m.UserID, &m.Field1, &m.Field2, &m.CreatedAt); err != nil {
            return nil, fmt.Errorf("failed to scan: %w", err)
        }
        models = append(models, m)
    }
    return models, rows.Err()
}
```

### 4. Update Store Interface

**Location**: `internal/api/server.go` - Add methods to Store interface

```go
type Store interface {
    // ... existing methods
    CreateMyNewModel(ctx context.Context, userID int64, field1 string, field2 int) (*MyNewModel, error)
    GetMyNewModelsByUser(ctx context.Context, userID int64) ([]MyNewModel, error)
}
```

### 5. Test Migration

```bash
# Delete database to test migration
rm noodexx.db

# Run application (migrations run automatically)
./noodexx

# Verify table was created
sqlite3 noodexx.db "SELECT name FROM sqlite_master WHERE type='table' AND name='my_new_table';"
```

---

## Adding a New UI Page

### 1. Create Template

**Location**: `web/templates/mypage.html`

```html
{{define "mypage-content"}}
<div class="container mx-auto px-4 py-8">
    <h1 class="text-3xl font-bold mb-6">My New Page</h1>
    
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <!-- Page content here -->
        <p>Content goes here</p>
    </div>
</div>
{{end}}
```

### 2. Create Handler

**Location**: `internal/api/handlers.go`

```go
func (s *Server) handleMyPage(w http.ResponseWriter, r *http.Request) {
    // Get user_id from context
    userID, err := auth.GetUserID(r.Context())
    if err != nil {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }
    
    // Get user's dark mode preference
    user, err := s.store.GetUserByID(r.Context(), userID)
    if err != nil {
        s.logger.Error("Failed to get user: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    // Prepare template data
    data := map[string]interface{}{
        "Title":                  "My Page",
        "Page":                   "mypage",
        "CloudProviderAvailable": s.providerManager.GetCloudProvider() != nil,
        "UIStyle":                s.uiStyle,
        "DarkMode":               user.DarkMode,
    }
    
    // Render template
    if err := s.templates.ExecuteTemplate(w, "base.html", data); err != nil {
        s.logger.Error("Failed to render template: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}
```

### 3. Register Route

**Location**: `internal/api/server.go`

```go
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
    // ... existing routes
    mux.HandleFunc("/mypage", s.handleMyPage)
}
```

### 4. Add Navigation Link

**Location**: `web/templates/base.html` - Add to sidebar

```html
<a href="/mypage" 
   class="sidebar-link"
   :class="{'active': window.location.pathname === '/mypage'}">
    <svg><!-- icon --></svg>
    <span>My Page</span>
</a>
```

---

## Adding a New Configuration Field

### 1. Update Config Struct

**Location**: `internal/config/config.go`

```go
type Config struct {
    // ... existing fields
    MyNewFeature MyNewFeatureConfig `json:"my_new_feature"`
}

type MyNewFeatureConfig struct {
    Enabled bool   `json:"enabled"`
    Setting string `json:"setting"`
}
```

### 2. Add Default Values

**Location**: `internal/config/config.go` in `Load()` function

```go
cfg := &Config{
    // ... existing defaults
    MyNewFeature: MyNewFeatureConfig{
        Enabled: false,
        Setting: "default",
    },
}
```

### 3. Add Validation

**Location**: `internal/config/config.go` in `Validate()` method

```go
func (c *Config) Validate() error {
    // ... existing validation
    
    // Validate new feature config
    if c.MyNewFeature.Enabled && c.MyNewFeature.Setting == "" {
        return fmt.Errorf("my_new_feature.setting is required when enabled")
    }
    
    return nil
}
```

### 4. Add Environment Variable Override

**Location**: `internal/config/config.go` in `applyEnvOverrides()` method

```go
func (c *Config) applyEnvOverrides() {
    // ... existing overrides
    
    if v := os.Getenv("NOODEXX_MY_NEW_FEATURE_ENABLED"); v != "" {
        c.MyNewFeature.Enabled = v == "true"
    }
    if v := os.Getenv("NOODEXX_MY_NEW_FEATURE_SETTING"); v != "" {
        c.MyNewFeature.Setting = v
    }
}
```

### 5. Update config.json.example

**Location**: `config.json.example`

```json
{
  "my_new_feature": {
    "enabled": false,
    "setting": "default"
  }
}
```

---

## Adding Provider Availability Check to UI

### Pattern for Disabling Features When Cloud Provider Unavailable

**In Handler**:
```go
cloudProviderAvailable := false
if s.providerManager != nil {
    cloudProviderAvailable = s.providerManager.GetCloudProvider() != nil
}

data := map[string]interface{}{
    "CloudProviderAvailable": cloudProviderAvailable,
    // ... other fields
}
```

**In Template**:
```html
<!-- Add data attribute -->
<div data-cloud-available="{{.CloudProviderAvailable}}">
    <!-- Disable button when unavailable -->
    <button id="cloudFeatureBtn" 
            {{if not .CloudProviderAvailable}}disabled{{end}}>
        Use Cloud Feature
    </button>
</div>

<!-- JavaScript to handle disabled state -->
<script>
document.addEventListener('DOMContentLoaded', function() {
    const cloudAvailable = document.querySelector('[data-cloud-available]')
        .getAttribute('data-cloud-available') === 'true';
    
    if (!cloudAvailable) {
        const btn = document.getElementById('cloudFeatureBtn');
        btn.disabled = true;
        btn.title = 'Cloud provider not configured';
        btn.classList.add('opacity-50', 'cursor-not-allowed');
    }
});
</script>
```

---

## Adding a New LLM Provider

### 1. Create Provider Implementation

**Location**: `internal/llm/myprovider.go`

```go
package llm

import (
    "context"
    "fmt"
    "noodexx/internal/logging"
)

type MyProvider struct {
    apiKey   string
    endpoint string
    logger   *logging.Logger
}

func NewMyProvider(cfg Config, logger *logging.Logger) (*MyProvider, error) {
    if cfg.MyProviderKey == "" {
        return nil, fmt.Errorf("my_provider API key is required")
    }
    
    return &MyProvider{
        apiKey:   cfg.MyProviderKey,
        endpoint: cfg.MyProviderEndpoint,
        logger:   logger,
    }, nil
}

func (p *MyProvider) Chat(ctx context.Context, messages []Message) (string, error) {
    // Implement chat completion
    // Call provider API
    // Return response
}

func (p *MyProvider) Embed(ctx context.Context, text string) ([]float32, error) {
    // Implement embedding generation
    // Call provider API
    // Return embedding vector
}

func (p *MyProvider) StreamChat(ctx context.Context, messages []Message) (<-chan string, <-chan error) {
    // Implement streaming chat
    // Return channels for response chunks and errors
}
```

### 2. Update Provider Factory

**Location**: `internal/llm/provider.go`

```go
func NewProvider(cfg Config, isLocal bool, logger *logging.Logger) (Provider, error) {
    switch cfg.Type {
    case "ollama":
        return NewOllamaProvider(cfg, logger)
    case "openai":
        return NewOpenAIProvider(cfg, logger)
    case "anthropic":
        return NewAnthropicProvider(cfg, logger)
    case "myprovider":
        return NewMyProvider(cfg, logger)
    default:
        return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
    }
}
```

### 3. Update Config

**Location**: `internal/config/config.go`

```go
type ProviderConfig struct {
    // ... existing fields
    MyProviderKey      string `json:"myprovider_key"`
    MyProviderEndpoint string `json:"myprovider_endpoint"`
}
```

### 4. Add Tests

**Location**: `internal/llm/myprovider_test.go`

```go
func TestMyProvider_Chat(t *testing.T) {
    // Test chat completion
}

func TestMyProvider_Embed(t *testing.T) {
    // Test embedding generation
}
```

---

## Adding User Preference Field

### 1. Add Column to Database

**Location**: `internal/store/migrations.go`

```go
{
    Version: 16,
    Description: "Add user preference field",
    Up: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            ALTER TABLE users 
            ADD COLUMN my_preference TEXT DEFAULT 'default'
        `)
        return err
    },
},
```

### 2. Update User Model

**Location**: `internal/store/models.go`

```go
type User struct {
    // ... existing fields
    MyPreference string `json:"my_preference"`
}
```

### 3. Add Update Method

**Location**: `internal/store/store.go`

```go
func (s *Store) UpdateUserPreference(ctx context.Context, userID int64, preference string) error {
    _, err := s.db.ExecContext(ctx, `
        UPDATE users 
        SET my_preference = ?
        WHERE id = ?
    `, preference, userID)
    if err != nil {
        return fmt.Errorf("failed to update preference: %w", err)
    }
    return nil
}
```

### 4. Add API Endpoint

**Location**: `internal/api/handlers.go`

```go
func (s *Server) handleUpdatePreference(w http.ResponseWriter, r *http.Request) {
    userID, _ := auth.GetUserID(r.Context())
    
    var req struct {
        Preference string `json:"preference"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    if err := s.store.UpdateUserPreference(r.Context(), userID, req.Preference); err != nil {
        http.Error(w, "Failed to update", http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
```

### 5. Add UI Control

**Location**: `web/templates/settings.html`

```html
<select id="myPreference" onchange="updatePreference(this.value)">
    <option value="default" {{if eq .User.MyPreference "default"}}selected{{end}}>Default</option>
    <option value="option1" {{if eq .User.MyPreference "option1"}}selected{{end}}>Option 1</option>
</select>

<script>
function updatePreference(value) {
    fetch('/api/update-preference', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({preference: value})
    });
}
</script>
```

---

## Next Steps

For more specific guidance, see:
- `04-testing-guide.md` - Testing strategies and patterns
- `05-ui-development.md` - Frontend development guide
- `06-troubleshooting.md` - Common issues and solutions
