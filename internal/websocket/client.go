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
	Hub       *Hub            `json:"-"`
	send      chan []byte     `json:"-"`
	Conn      *websocket.Conn `json:"-"`
	Searching bool            `json:"-"`

	SessionID   string   `json:"session_id"`
	ChatPartner *Client  `json:"-"`
	ChatType    string   `json:"chat_type"`
	IsStrict    bool     `json:"is_strict"`
	Interests   []string `json:"interests"`
}

func (c Client) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("session_id", c.SessionID),
		slog.String("chat_type", c.ChatType),
		slog.Any("interests", c.Interests),
	)
}

func NewClient(sessionID string, chatType string, isStrict bool, interests []string, hub *Hub) *Client {
	return &Client{
		Hub:       hub,
		SessionID: sessionID,
		ChatType:  chatType,
		IsStrict:  isStrict,
		Interests: interests,
		send:      make(chan []byte, 256),
	}
}

func (client *Client) IsActive() bool {
	var err error
	if client.Conn == nil {
		return false
	}
	err = client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(pongWait))

	return err == nil
}

func (client *Client) ReadPump() {
	defer func() {
		err := client.Hub.UnregisterClient(client)
		if err != nil {
			client.Hub.logger.Error("error in ReadPump, UnregisterClient", slog.String("error", err.Error()))
		}
		if r := recover(); r != nil {
			client.Hub.logger.Error("panic in Client.ReadPump", slog.Any("error", r))
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
					client.Hub.logger.Error("ReadPump: can't read from connection ", slog.String("error", err.Error()))
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
			client.Hub.logger.Error("panic in Client.WritePump", slog.Any("error", r))
		}
	}()

	for {
		select {
		case <-client.Hub.ctx.Done():
			client.Hub.logger.Info("closing WritePump (cancellation)")
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

	if client.ChatPartner != nil {
		return fmt.Errorf("(client.Connect) Client already connected to a partner")
	}

	if partner.ChatPartner != nil {
		return fmt.Errorf("(client.Connect) Partner already connected to another client")
	}

	client.ChatPartner = partner
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
