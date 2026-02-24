package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketHub_NewWebSocketHub(t *testing.T) {
	hub := NewWebSocketHub()

	if hub == nil {
		t.Fatal("NewWebSocketHub returned nil")
	}

	if hub.clients == nil {
		t.Error("clients map not initialized")
	}

	if hub.broadcast == nil {
		t.Error("broadcast channel not initialized")
	}

	if hub.register == nil {
		t.Error("register channel not initialized")
	}

	if hub.unregister == nil {
		t.Error("unregister channel not initialized")
	}
}

func TestWebSocketHub_RegisterAndUnregister(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	// Create a test WebSocket connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade: %v", err)
		}
		hub.register <- conn
		time.Sleep(50 * time.Millisecond) // Give time for registration
	}))
	defer server.Close()

	// Connect to the test server
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond) // Give time for registration

	// Verify client is registered
	hub.mu.RLock()
	clientCount := len(hub.clients)
	hub.mu.RUnlock()

	if clientCount != 1 {
		t.Errorf("Expected 1 client, got %d", clientCount)
	}
}

func TestWebSocketHub_Broadcast(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade: %v", err)
		}
		hub.register <- conn
	}))
	defer server.Close()

	// Connect to the test server
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond) // Give time for registration

	// Broadcast a message
	hub.Broadcast("test", "Hello, WebSocket!")

	// Read the message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	// Verify message content by checking if it contains expected fields
	messageStr := string(message)
	if !strings.Contains(messageStr, `"type":"test"`) {
		t.Errorf("Message missing type field: %s", messageStr)
	}
	if !strings.Contains(messageStr, `"message":"Hello, WebSocket!"`) {
		t.Errorf("Message missing message field: %s", messageStr)
	}
}

func TestServer_handleWebSocket(t *testing.T) {
	// Create a minimal server for testing
	hub := NewWebSocketHub()
	go hub.Run()

	server := &Server{
		wsHub: hub,
	}

	// Create a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer ts.Close()

	// Connect to the WebSocket endpoint
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial WebSocket: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond) // Give time for registration

	// Verify client is registered
	hub.mu.RLock()
	clientCount := len(hub.clients)
	hub.mu.RUnlock()

	if clientCount != 1 {
		t.Errorf("Expected 1 client registered, got %d", clientCount)
	}

	// Close connection and verify unregistration
	conn.Close()
	time.Sleep(100 * time.Millisecond) // Give time for unregistration

	hub.mu.RLock()
	clientCount = len(hub.clients)
	hub.mu.RUnlock()

	if clientCount != 0 {
		t.Errorf("Expected 0 clients after disconnect, got %d", clientCount)
	}
}
