package websocket

import (
	"bytes"
	"context"
	"log/slog"

	"github.com/SegniAdebaGodsSon/ui/templates"
)

type Hub struct {
	OnlineClients *OnlineClients
	Register      chan *Client
	Unregister    chan *Client
	Recieve       chan *Message
}

func (h *Hub) Run() {
	slog.Info("Hub running")
	for {
		select {
		case client := <-h.Register:
			if client != nil {
				slog.Info("client joined chat", "client", client.SessionID)

				partner := h.OnlineClients.FindMatchingClient(client.SessionID)
				html := bytes.NewBuffer(nil)
				if partner == nil {
					component := templates.ConnectionStatusNoClientsFound()
					component.Render(context.Background(), html)
					client.SendMessage(html.Bytes())
				} else {
					component := templates.ConnectionStatusConnected()
					component.Render(context.Background(), html)

					client.Connect(partner)

					client.SendMessage(html.Bytes())
					partner.SendMessage(html.Bytes())
				}
			}

		case client := <-h.Unregister:
			slog.Info("client left chat", "client", client.SessionID)
			h.OnlineClients.DeleteClient(client.SessionID)

			if partner := client.ChatPartner; partner != nil {
				client.Disconnect()
				partner.Disconnect()

				// send a connected peer that they're disconnected
				html := bytes.NewBuffer(nil)
				component := templates.ConnectionStatusDisconnected()
				component.Render(context.Background(), html)

				partner.SendMessage(html.Bytes())
				client.SendMessage(html.Bytes())
			}

		case message := <-h.Recieve:
			message_type := message.Type
			switch message_type {
			case "chat_message":
				client := message.From
				if client == nil {
					continue
				}

				partner := message.From.ChatPartner
				if partner == nil {
					continue
				}

				clientMessageComponent := templates.ChatBubble(message.ChatMessage, true)
				partnerMessageComponent := templates.ChatBubble(message.ChatMessage, false)

				html := bytes.NewBuffer(nil)
				clientMessageComponent.Render(context.Background(), html)
				client.SendMessage(html.Bytes())

				html = bytes.NewBuffer(nil)
				partnerMessageComponent.Render(context.Background(), html)
				partner.SendMessage(html.Bytes())
			case "typing":
				client := message.From
				if client == nil {
					continue
				}

				partner := message.From.ChatPartner
				if partner == nil {
					continue
				}

				typingMessageComponent := templates.StrangerTyping()
				html := bytes.NewBuffer(nil)
				typingMessageComponent.Render(context.Background(), html)
				partner.SendMessage(html.Bytes())
			}

		}
	}
}
