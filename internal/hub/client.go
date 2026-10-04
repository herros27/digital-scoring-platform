package hub

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID         string
	Role       Role
	SeatNumber int
	ArenaID    string
	conn       *websocket.Conn
	send       chan Message
	hub        *Hub
}

// NewClient creates a client instance (useful for testing and programmatic clients)
func NewClient(id string, role Role, arenaID string) (*Client, <-chan Message) {
	ch := make(chan Message, 256)
	return &Client{
		ID:      id,
		Role:    role,
		ArenaID: arenaID,
		send:    ch,
	}, ch
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for msg := range c.send {
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("marshal error: %v", err)
			continue
		}
		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		msg.From = c.ID

		c.hub.inbound <- inboundMessage{
			client: c,
			msg:    msg,
		}
	}
}
