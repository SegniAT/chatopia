package websocket

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/SegniAdebaGodsSon/ui/templates"
	"github.com/google/uuid"
)

type Hub struct {
	OnlineClients *OnlineClients
	Register      chan *Client
	Unregister    chan *Client
	Recieve       chan *Message
}

func (h *Hub) Run() {
	slog.Info("Hub running")

	HtmlConnectionReady := bytes.NewBuffer(nil)
	templates.ConnectionStatusReady().Render(context.Background(), HtmlConnectionReady)

	HtmlConnectionStatusSearching := bytes.NewBuffer(nil)
	templates.ConnectionStatusSearching().Render(context.Background(), HtmlConnectionStatusSearching)

	HtmlConnectionStatusConnected := bytes.NewBuffer(nil)
	templates.ConnectionStatusConnected().Render(context.Background(), HtmlConnectionStatusConnected)

	HtmlConnectionStatusDisconnected := bytes.NewBuffer(nil)
	templates.ConnectionStatusDisconnected(false).Render(context.Background(), HtmlConnectionStatusDisconnected)

	HtmlConnectionStatusDisconnectedAutoReconnect := bytes.NewBuffer(nil)
	templates.ConnectionStatusDisconnected(true).Render(context.Background(), HtmlConnectionStatusDisconnectedAutoReconnect)

	HtmlConnectionStatusNoClientsFound := bytes.NewBuffer(nil)
	templates.ConnectionStatusNoClientsFound().Render(context.Background(), HtmlConnectionStatusNoClientsFound)

	HtmlStrangerTyping := bytes.NewBuffer(nil)
	templates.StrangerTyping().Render(context.Background(), HtmlStrangerTyping)

	for {
		select {
		case client := <-h.Register:
			if client == nil {
				continue
			}
			slog.Info("client joined chat", "client", client.SessionID)

			partner := h.OnlineClients.FindMatchingClient(client.SessionID)
			if partner == nil {
				client.SendMessage(HtmlConnectionStatusNoClientsFound.Bytes())
			} else {
				err := client.Connect(partner)
				if err != nil {
					client.SendMessage(HtmlConnectionStatusDisconnected.Bytes())
					continue
				}

				client.SendMessage(HtmlConnectionStatusConnected.Bytes())
				partner.SendMessage(HtmlConnectionStatusConnected.Bytes())

				if client.ChatType == "video" {
					callerPeerId := uuid.NewString()
					partnerPeerId := uuid.NewString()

					callerMessage := Message{
						Type:        "PEER_ID_PARTNER",
						ChatMessage: fmt.Sprintf("{\"id\":\"%s\", \"partner_id\":\"%s\"}", callerPeerId, partnerPeerId),
					}
					callerMessageHtml, _ := callerMessage.Encode()

					partnerMessage := Message{
						Type:        "PEER_ID_CALLER",
						ChatMessage: fmt.Sprintf("{\"id\":\"%s\", \"caller_id\":\"%s\"}", partnerPeerId, callerPeerId),
					}
					partnerMessageHtml, _ := partnerMessage.Encode()

					client.SendMessage(callerMessageHtml)
					partner.SendMessage(partnerMessageHtml)
				}
			}

		case client := <-h.Unregister:
			if client == nil {
				continue
			}
			slog.Info("client left chat", "client", client.SessionID)
			h.OnlineClients.DeleteClient(client.SessionID)

			partner := client.Disconnect()

			if client.AutoReconnect {
				client.SendMessage(HtmlConnectionStatusDisconnectedAutoReconnect.Bytes())
			} else {
				client.SendMessage(HtmlConnectionStatusDisconnected.Bytes())
			}

			if partner != nil {
				partner.Disconnect()

				if partner.AutoReconnect {
					partner.SendMessage(HtmlConnectionStatusDisconnectedAutoReconnect.Bytes())
				} else {
					partner.SendMessage(HtmlConnectionStatusDisconnected.Bytes())
				}
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
				partner.SendMessage(HtmlStrangerTyping.Bytes())

			case "new_connection":
				client := message.From
				if client == nil {
					continue
				}

				// disconnect
				partner := client.Disconnect()
				if partner != nil {
					if partner.AutoReconnect {
						partner.SendMessage(HtmlConnectionStatusDisconnectedAutoReconnect.Bytes())
					} else {
						partner.SendMessage(HtmlConnectionStatusDisconnected.Bytes())
					}

					if !(message.Headers.HXTrigger == "connection_status") {
						if client.AutoReconnect {
							client.SendMessage(HtmlConnectionStatusDisconnectedAutoReconnect.Bytes())
						} else {
							client.SendMessage(HtmlConnectionStatusDisconnected.Bytes())
						}
					}
				}

				// search and connect
				partner = h.OnlineClients.FindMatchingClient(client.SessionID)
				if partner == nil {
					client.SendMessage(HtmlConnectionStatusNoClientsFound.Bytes())
				} else {
					client.Connect(partner)

					client.SendMessage(HtmlConnectionStatusConnected.Bytes())
					partner.SendMessage(HtmlConnectionStatusConnected.Bytes())

					// send CALL_OFFER
					/*
						if client.ChatType == "video" {
							offer := Message{
								Type:        "PEER",
								ChatMessage: "CALL_OFFER",
							}
							offerBytes, err := offer.Encode()
							if err != nil {
								slog.Error("HUB (CALL_OFFER)", "error", err.Error())
								continue
							}

							partner.SendMessage(offerBytes)
						}
					*/
				}

			case "auto_connect":
				client := message.From
				client.AutoReconnect = !client.AutoReconnect

				html := bytes.NewBuffer(nil)
				templates.AutoConnect(client.AutoReconnect).Render(context.Background(), html)
				client.SendMessage(html.Bytes())
			}

		}
	}
}
