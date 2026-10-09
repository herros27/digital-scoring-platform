package hub

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

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

// Keepalive WebSocket: server mengirim ping tiap pingPeriod dan memutus koneksi yang tidak membalas pong dalam
// pongWait. Tanpa ini tablet yang kehilangan jaringan tanpa menutup soket tetap tercatat "online" di panel wasit
// dan menghalangi sambung ulang kursi yang sama. Dapat diubah lewat WS_PONG_WAIT_MS (ping = 9/10 dari itu).
var (
	pongWait   = envDuration("WS_PONG_WAIT_MS", 60*time.Second)
	pingPeriod = pongWait * 9 / 10
	writeWait  = 10 * time.Second
)

func envDuration(key string, def time.Duration) time.Duration {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return time.Duration(v) * time.Millisecond
	}
	return def
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("marshal error: %v", err)
				continue
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

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
