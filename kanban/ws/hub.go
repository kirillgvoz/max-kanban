package ws

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"sync"
	"time"
)

type Ticket struct {
	UserID    int64
	OrgID     int64
	ExpiresAt time.Time
}

type Hub struct {
	rooms   map[int64]map[*Client]struct{}
	tickets map[string]Ticket
	mu      sync.RWMutex
}

type Client struct {
	Hub       *Hub
	UserID    int64
	OrgID     int64
	Send      chan []byte
	CloseConn func()
	closeOnce sync.Once
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

var DefaultHub = NewHub()

func NewHub() *Hub {
	return &Hub{
		rooms:   make(map[int64]map[*Client]struct{}),
		tickets: make(map[string]Ticket),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	if h.rooms[client.OrgID] == nil {
		h.rooms[client.OrgID] = make(map[*Client]struct{})
	}
	h.rooms[client.OrgID][client] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	if clients := h.rooms[client.OrgID]; clients != nil {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.rooms, client.OrgID)
		}
	}
	h.mu.Unlock()
	client.closeOnce.Do(func() {
		if client.CloseConn != nil {
			client.CloseConn()
		}
	})
}

func (h *Hub) Broadcast(orgID int64, message Message) {
	encoded := mustMarshal(message)
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.rooms[orgID]))
	for client := range h.rooms[orgID] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.Send <- encoded:
		default:
			h.Unregister(client)
		}
	}
}

func (h *Hub) IssueTicket(userID, orgID int64) string {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return ""
	}
	ticket := base64.RawURLEncoding.EncodeToString(buffer)
	h.mu.Lock()
	h.tickets[ticket] = Ticket{UserID: userID, OrgID: orgID, ExpiresAt: time.Now().Add(30 * time.Second)}
	h.cleanupTicketsLocked()
	h.mu.Unlock()
	return ticket
}

func (h *Hub) ConsumeTicket(raw string) (Ticket, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ticket, ok := h.tickets[raw]
	if ok {
		delete(h.tickets, raw)
	}
	if !ok || time.Now().After(ticket.ExpiresAt) || ticket.UserID <= 0 || ticket.OrgID <= 0 {
		return Ticket{}, false
	}
	return ticket, true
}

func (h *Hub) Count(orgID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[orgID])
}

func (h *Hub) Total() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, clients := range h.rooms {
		total += len(clients)
	}
	return total
}

func (h *Hub) cleanupTicketsLocked() {
	now := time.Now()
	for key, ticket := range h.tickets {
		if now.After(ticket.ExpiresAt) {
			delete(h.tickets, key)
		}
	}
}

func logConnection(message string, orgID int64) {
	log.Printf("%s org=%d", message, orgID)
}
