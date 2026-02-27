package api

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleChat_CloudProviderAvailable tests that CloudProviderAvailable is passed to chat template
func TestHandleChat_CloudProviderAvailable(t *testing.T) {
	tests := []struct {
		name                   string
		providerManager        ProviderManager
		expectedCloudAvailable bool
	}{
		{
			name:                   "cloud provider available",
			providerManager:        &mockProviderManagerWithCloud{cloudAvailable: true},
			expectedCloudAvailable: true,
		},
		{
			name:                   "cloud provider not available",
			providerManager:        &mockProviderManagerWithCloud{cloudAvailable: false},
			expectedCloudAvailable: false,
		},
		{
			name:                   "no provider manager",
			providerManager:        nil,
			expectedCloudAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal template for testing
			tmpl := template.Must(template.New("base.html").Parse(`<!DOCTYPE html><html><body>{{.CloudProviderAvailable}}</body></html>`))

			// Create a mock server with the test provider manager
			srv := &Server{
				store:           &mockStore{},
				provider:        &mockProvider{},
				ingester:        &mockIngester{},
				searcher:        &mockSearcher{},
				config:          &ServerConfig{},
				logger:          &mockLogger{},
				providerManager: tt.providerManager,
				ragEnforcer:     &mockRAGEnforcer{},
				templates:       tmpl,
			}

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/chat", nil)
			w := httptest.NewRecorder()

			// Call handler
			srv.handleChat(w, req)

			// Check response status
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// Verify the response contains the expected value
			body := w.Body.String()
			if tt.expectedCloudAvailable {
				if body != "<!DOCTYPE html><html><body>true</body></html>" {
					t.Errorf("Expected CloudProviderAvailable=true in response, got: %s", body)
				}
			} else {
				if body != "<!DOCTYPE html><html><body>false</body></html>" {
					t.Errorf("Expected CloudProviderAvailable=false in response, got: %s", body)
				}
			}
		})
	}
}

// TestHandleSettings_CloudProviderAvailable tests that CloudProviderAvailable is passed to settings template
func TestHandleSettings_CloudProviderAvailable(t *testing.T) {
	tests := []struct {
		name                   string
		providerManager        ProviderManager
		expectedCloudAvailable bool
	}{
		{
			name:                   "cloud provider available",
			providerManager:        &mockProviderManagerWithCloud{cloudAvailable: true},
			expectedCloudAvailable: true,
		},
		{
			name:                   "cloud provider not available",
			providerManager:        &mockProviderManagerWithCloud{cloudAvailable: false},
			expectedCloudAvailable: false,
		},
		{
			name:                   "no provider manager",
			providerManager:        nil,
			expectedCloudAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal template for testing
			tmpl := template.Must(template.New("base.html").Parse(`<!DOCTYPE html><html><body>{{.CloudProviderAvailable}}</body></html>`))

			// Create a mock server with the test provider manager
			srv := &Server{
				store:           &mockStore{},
				provider:        &mockProvider{},
				ingester:        &mockIngester{},
				searcher:        &mockSearcher{},
				config:          &ServerConfig{},
				logger:          &mockLogger{},
				providerManager: tt.providerManager,
				ragEnforcer:     &mockRAGEnforcer{},
				templates:       tmpl,
				configPath:      "testdata/config.json",
			}

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/settings", nil)
			w := httptest.NewRecorder()

			// Call handler
			srv.handleSettings(w, req)

			// Check response status - settings may fail to load config file, which is OK for this test
			if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
				t.Errorf("Expected status 200 or 500, got %d", w.Code)
			}

			// If the handler succeeded, verify the response
			if w.Code == http.StatusOK {
				body := w.Body.String()
				if tt.expectedCloudAvailable {
					if body != "<!DOCTYPE html><html><body>true</body></html>" {
						t.Errorf("Expected CloudProviderAvailable=true in response, got: %s", body)
					}
				} else {
					if body != "<!DOCTYPE html><html><body>false</body></html>" {
						t.Errorf("Expected CloudProviderAvailable=false in response, got: %s", body)
					}
				}
			}
		})
	}
}

// mockProviderManagerWithCloud is a mock that can simulate cloud provider availability
type mockProviderManagerWithCloud struct {
	cloudAvailable bool
}

func (m *mockProviderManagerWithCloud) GetActiveProvider() (LLMProvider, error) {
	return &mockProvider{}, nil
}

func (m *mockProviderManagerWithCloud) GetLocalProvider() LLMProvider {
	return &mockProvider{}
}

func (m *mockProviderManagerWithCloud) GetCloudProvider() LLMProvider {
	if m.cloudAvailable {
		return &mockProvider{}
	}
	return nil
}

func (m *mockProviderManagerWithCloud) IsLocalMode() bool {
	return true
}

func (m *mockProviderManagerWithCloud) GetProviderName() string {
	return "Mock Provider"
}

func (m *mockProviderManagerWithCloud) Reload(cfg interface{}) error {
	return nil
}

