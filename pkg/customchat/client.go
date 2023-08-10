package customchat

import (
	"bytes"
	"log"
	"time"

	"github.com/fasthttp/websocket"
)

// CustomClient represents a connected client.
type CustomClient struct {
	Hub  *CustomHub
	Conn *websocket.Conn
	Send chan []byte
}

const (
	writeInterval  = 10 * time.Second
	pingInterval   = (pongInterval * 9) / 10
	pongInterval   = 40 * time.Second
	maxMessageSize = 512
)

var (
	newLine = []byte("\n")
	space   = []byte(" ")
)

func (c *CustomClient) handlePong() {
	c.Conn.SetReadDeadline(time.Now().Add(pongInterval))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongInterval))
		return nil
	})
}

func (c *CustomClient) readLoop() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.handlePong()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, newLine, space, -1))
		c.Hub.Broadcast <- message
	}
}

func (c *CustomClient) sendPing(pingTicker *time.Ticker) {
	c.Conn.SetWriteDeadline(time.Now().Add(writeInterval))
	if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		return
	}
}

func (c *CustomClient) writeLoop(pingTicker *time.Ticker) {
	defer func() {
		pingTicker.Stop()
		c.Conn.Close()
	}()

	c.writeMessages(pingTicker)
}

func (c *CustomClient) writeMessages(pingTicker *time.Ticker) {
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				return
			}
			c.writeMessageAndHandleMore(c.Conn, message)
		case <-pingTicker.C:
			c.sendPing(pingTicker)
		}
	}
}

func (c *CustomClient) writeMessageAndHandleMore(conn *websocket.Conn, message []byte) {
	conn.SetWriteDeadline(time.Now().Add(writeInterval))
	writer, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return
	}
	defer writer.Close()

	writer.Write(message)

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				return
			}
			writer.Write(newLine)
			writer.Write(message)
		default:
			return
		}
	}
}

// NewCustomClient creates a new CustomClient instance.
func NewCustomClient(hub *CustomHub, conn *websocket.Conn) *CustomClient {
	return &CustomClient{
		Hub:  hub,
		Conn: conn,
		Send: make(chan []byte),
	}
}

// NewPeerChatConnection creates a new PeerChatConnection instance and starts communication goroutines.
func NewPeerChatConnection(conn *websocket.Conn, hub *CustomHub) {
	client := NewCustomClient(hub, conn)
	client.Hub.Register <- client

	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	go client.writeLoop(pingTicker)
	go client.readLoop()
}
