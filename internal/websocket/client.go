package websocket

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type Client struct {
	Hub       *Hub
	send      chan []byte
	Conn      *websocket.Conn
	Searching bool

	SessionID   string
	ChatPartner *Client
	ChatType    string
	Interests   []string
}

func NewClient(sessionID string, chatType string, interests []string, hub *Hub) *Client {
	return &Client{
		Hub:       hub,
		SessionID: sessionID,
		ChatType:  chatType,
		Interests: interests,
		send:      make(chan []byte, 256),
	}
}

func (client *Client) ReadPump() {
	defer func() {
		client.Hub.Unregister <- client
		if r := recover(); r != nil {
			slog.Error("panic in Client.ReadPump")
		}
	}()

	client.Conn.SetReadLimit(maxMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-client.Hub.ctx.Done():
			return
		default:
			message := &Message{}
			err := client.Conn.ReadJSON(message)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					slog.Error("(readPump)", "error: ", err.Error())
				}
				break
			}

			message.From = client
			client.Hub.Recieve <- message
		}
	}
}

func (client *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()

		if r := recover(); r != nil {
			slog.Error("panic in Client.WritePump")
		}
	}()

	for {
		select {
		case <-client.Hub.ctx.Done():
			slog.Info("closing WritePump (cancellation)")
			return
		case message, ok := <-client.send:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// the hub closed the channel
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (client *Client) SendMessage(message []byte) {
	client.send <- message
}

func (client *Client) Connect(partner *Client) error {
	if partner == nil {
		return fmt.Errorf("(client.Connect) Partner cannot be nil")
	}

	if client.SessionID == partner.SessionID {
		return fmt.Errorf("(client.Connect) Client cannot connect with itself")
	}

	client.Disconnect()
	client.ChatPartner = partner

	partner.Disconnect()
	partner.ChatPartner = client
	return nil
}

func (client *Client) Disconnect() *Client {
	partner := client.ChatPartner
	client.ChatPartner = nil
	if partner != nil {
		partner.ChatPartner = nil
	}
	return partner
}
