package hub

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/herros27/digital-scoring/internal/metrics"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type inboundMessage struct {
	client *Client
	msg    Message
}

type Hub struct {
	mu         sync.RWMutex
	ArenaID    string
	clients    map[*Client]bool
	inbound    chan inboundMessage
	register   chan *Client
	unregister chan *Client
	// callback ke orchestrator
	OnMessage func(arenaID string, client *Client, msg Message)
	// onPresence dipanggil (di luar kunci hub) setiap ada klien yang terhubung/terputus
	onPresence func()
}

// Presence ringkasan perangkat yang sedang terhubung di arena.
type Presence struct {
	JudgeSeats  map[int]bool // nomor kursi juri yang online
	Referees    int
	Admins      int
	Scoreboards int
}

// Presence menghitung perangkat yang terhubung saat ini.
func (h *Hub) Presence() Presence {
	h.mu.RLock()
	defer h.mu.RUnlock()
	p := Presence{JudgeSeats: make(map[int]bool)}
	for c := range h.clients {
		switch {
		case c.Role.IsJudge():
			p.JudgeSeats[c.SeatNumber] = true
		case c.Role == RoleReferee:
			p.Referees++
		case c.Role == RoleAdmin:
			p.Admins++
		case c.Role == RoleScoreboard:
			p.Scoreboards++
		}
	}
	return p
}

// SetOnPresence memasang callback perubahan kehadiran (aman dipanggil saat hub berjalan).
func (h *Hub) SetOnPresence(fn func()) {
	h.mu.Lock()
	h.onPresence = fn
	h.mu.Unlock()
}

func (h *Hub) notifyPresence() {
	h.mu.RLock()
	fn := h.onPresence
	h.mu.RUnlock()
	if fn != nil {
		fn()
	}
}

func NewHub(arenaID string) *Hub {
	return &Hub{
		ArenaID:    arenaID,
		clients:    make(map[*Client]bool),
		inbound:    make(chan inboundMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[hub:%s] client connected: %s (%s)", h.ArenaID, client.ID, client.Role)
			metrics.IncWSConnections(h.ArenaID, string(client.Role))
			h.notifyPresence()

		case client := <-h.unregister:
			h.mu.Lock()
			removed := false
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				removed = true
				log.Printf("[hub:%s] client disconnected: %s", h.ArenaID, client.ID)
				metrics.DecWSConnections(h.ArenaID, string(client.Role))
			}
			h.mu.Unlock()
			if removed {
				h.notifyPresence()
			}

		case inbound := <-h.inbound:
			// judge tidak bisa lihat input judge lain
			if h.OnMessage != nil {
				h.OnMessage(h.ArenaID, inbound.client, inbound.msg)
			}
		}
	}
}

// Broadcast ke semua client di arena
func (h *Hub) Broadcast(msg Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		select {
		case client.send <- msg:
		default:
			close(client.send)
			delete(h.clients, client)
			metrics.DecWSConnections(h.ArenaID, string(client.Role))
		}
	}
}

// Send ke role tertentu saja
func (h *Hub) SendToRole(role Role, msg Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		if client.Role == role {
			select {
			case client.send <- msg:
			default:
				close(client.send)
				delete(h.clients, client)
				metrics.DecWSConnections(h.ArenaID, string(client.Role))
			}
		}
	}
}

// Send ke client tertentu
func (h *Hub) SendToClient(clientID string, msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.ID == clientID {
			select {
			case client.send <- msg:
			default:
			}
			return
		}
	}
}

// Register adds a client directly to the hub (useful for testing or embedded clients)
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
}

// HandleWS — HTTP handler untuk upgrade ke WebSocket
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request, clientID string, role Role) {
	h.HandleWSWithSeat(w, r, clientID, role, 0)
}

// HandleWSWithSeat — HTTP handler untuk upgrade ke WebSocket dengan nomor kursi
func (h *Hub) HandleWSWithSeat(w http.ResponseWriter, r *http.Request, clientID string, role Role, seatNumber int) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := &Client{
		ID:         clientID,
		Role:       role,
		SeatNumber: seatNumber,
		ArenaID:    h.ArenaID,
		conn:       conn,
		send:       make(chan Message, 256),
		hub:        h,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}
