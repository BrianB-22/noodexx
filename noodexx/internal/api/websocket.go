package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

// NewWebSocketHub creates a hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Run starts the hub's event loop
func (h *WebSocketHub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					conn.Close()
					delete(h.clients, conn)
				}
			}
			h.mu.Unlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHub) Broadcast(eventType, message string) {
	data := map[string]string{
		"type":    eventType,
		"message": message,
	}

	jsonData, _ := json.Marshal(data)
	h.broadcast <- jsonData
}

// handleWebSocket upgrades HTTP to WebSocket
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // In production, validate origin
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.wsHub.register <- conn

	// Read loop (handle client messages if needed)
	go func() {
		defer func() {
			s.wsHub.unregister <- conn
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}
