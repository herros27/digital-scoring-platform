package hub

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type inboundMessage struct {
	client *Client
	msg    Message
}

type Hub struct {
	ArenaID    string
	clients    map[*Client]bool
	inbound    chan inboundMessage
	register   chan *Client
	unregister chan *Client
	// callback ke orchestrator
	OnMessage func(arenaID string, client *Client, msg Message)
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
			h.clients[client] = true
			log.Printf("[hub:%s] client connected: %s (%s)", h.ArenaID, client.ID, client.Role)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("[hub:%s] client disconnected: %s", h.ArenaID, client.ID)
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
	for client := range h.clients {
		select {
		case client.send <- msg:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

// Send ke role tertentu saja
func (h *Hub) SendToRole(role Role, msg Message) {
	for client := range h.clients {
		if client.Role == role {
			select {
			case client.send <- msg:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// Send ke client tertentu
func (h *Hub) SendToClient(clientID string, msg Message) {
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