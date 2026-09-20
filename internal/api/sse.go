package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/model"
)

// SSEBroker manages SSE client connections and event broadcasting.
type SSEBroker struct {
	clients    map[chan []byte]bool
	register   chan chan []byte
	unregister chan chan []byte
	broadcast  chan []byte
	done       chan struct{}
	mu         sync.RWMutex
}

// NewSSEBroker creates a new SSE broker.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients:    make(map[chan []byte]bool),
		register:   make(chan chan []byte),
		unregister: make(chan chan []byte),
		broadcast:  make(chan []byte, 100),
		done:       make(chan struct{}),
	}
}

// Start begins the broker's event loop.
func (b *SSEBroker) Start() {
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-b.done:
			return

		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = true
			b.mu.Unlock()
			log.Printf("SSE client connected (%d total)", len(b.clients))

		case client := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				close(client)
			}
			b.mu.Unlock()
			log.Printf("SSE client disconnected (%d total)", len(b.clients))

		case msg := <-b.broadcast:
			b.mu.RLock()
			for client := range b.clients {
				select {
				case client <- msg:
				default:
					// Client buffer full, skip
				}
			}
			b.mu.RUnlock()

		case <-heartbeatTicker.C:
			b.sendHeartbeat()
		}
	}
}

// Stop shuts down the broker.
func (b *SSEBroker) Stop() {
	close(b.done)
	b.mu.Lock()
	for client := range b.clients {
		close(client)
	}
	b.clients = make(map[chan []byte]bool)
	b.mu.Unlock()
}

// Subscribe registers a new client and returns their message channel.
func (b *SSEBroker) Subscribe() chan []byte {
	client := make(chan []byte, 10)
	b.register <- client
	return client
}

// Unsubscribe removes a client.
func (b *SSEBroker) Unsubscribe(client chan []byte) {
	b.unregister <- client
}

// Broadcast sends an event to all connected clients.
func (b *SSEBroker) Broadcast(event model.Event) {
	data, err := json.Marshal(event.Data)
	if err != nil {
		log.Printf("SSE marshal error: %v", err)
		return
	}

	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, data)
	b.broadcast <- []byte(msg)
}

// sendHeartbeat sends a heartbeat event to all clients.
func (b *SSEBroker) sendHeartbeat() {
	b.Broadcast(model.NewHeartbeat())
}

// handleEvents handles GET /api/v1/events (SSE endpoint).
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// Check if SSE is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "SSE_NOT_SUPPORTED",
			"Streaming not supported")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to events
	client := s.sse.Subscribe()
	defer s.sse.Unsubscribe(client)

	// Send initial connection event
	// A failed first write means the client already went away; there is nothing to stream to.
	if _, err := fmt.Fprintf(w, "event: connected\ndata: {\"message\":\"Connected to Gastown Viewer Intent\"}\n\n"); err != nil {
		return
	}
	flusher.Flush()

	// Listen for events or client disconnect
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-client:
			if !ok {
				return
			}
			_, _ = w.Write(msg)
			flusher.Flush()
		}
	}
}

// NotifyIssueCreated broadcasts an issue_created event.
func (s *Server) NotifyIssueCreated(id, title string, status model.Status) {
	s.sse.Broadcast(model.NewIssueCreatedEvent(id, title, status))
}

// NotifyIssueUpdated broadcasts an issue_updated event.
func (s *Server) NotifyIssueUpdated(id string, status, previousStatus model.Status) {
	s.sse.Broadcast(model.NewIssueUpdatedEvent(id, status, previousStatus))
}
