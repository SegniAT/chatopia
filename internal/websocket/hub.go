package websocket

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/SegniAdebaGodsSon/ui/templates"
	"github.com/a-h/templ"
	"github.com/google/uuid"
)

const (
	SEARCH_RETRIES    = 5
	SEARCH_RETRY_WAIT = time.Second * 3
	CLEANUP_PERIOD    = time.Second * 10
)

type Hub struct {
	OnlineClients *OnlineClients
	Register      chan *Client
	Unregister    chan *Client
	Recieve       chan *Message
	ctx           context.Context
	wg            sync.WaitGroup
	logger        *slog.Logger
}

func NewHub(ctx context.Context, logger *slog.Logger) *Hub {
	return &Hub{
		OnlineClients: NewOnlineClientsStore(),
		Recieve:       make(chan *Message),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		ctx:           ctx,
		logger:        logger,
	}
}

func (h *Hub) Run() {
	defer func() {
		if r := recover(); r != nil {
			h.logger.ErrorContext(context.Background(), "panic in Hub.Run", slog.Any("error", r))
		}
	}()

	h.logger.InfoContext(context.Background(), "Hub has started running")

	go h.cleanDisconnectedClients()

	HtmlTemplates := preloadTemplates()

	for {
		select {
		case <-h.ctx.Done():
			h.wg.Wait()
			return

		case client := <-h.Register:
			h.wg.Add(1)
			go h.handleRegistration(client, HtmlTemplates)

		case client := <-h.Unregister:
			h.wg.Add(1)
			go h.handleUnregistration(client, HtmlTemplates)

		case message := <-h.Recieve:
			h.wg.Add(1)
			go h.handleMessage(message, HtmlTemplates)
		}
	}
}

func (h *Hub) cleanDisconnectedClients() {
	go func() {
		ticker := time.NewTicker(CLEANUP_PERIOD)
		defer ticker.Stop()

		for {
			select {
			case <-h.ctx.Done():
				return
			case <-ticker.C:
				h.checkClients()
			}
		}
	}()
}

func (h *Hub) checkClients() {
	h.OnlineClients.Range(func(key, value any) bool {
		client, ok := value.(*Client)
		if !ok {
			return true
		}

		if !client.IsActive() {
			h.OnlineClients.DeleteClient(client.SessionID)
		}

		return true
	})
}

func (h *Hub) connectToClient(client *Client, HtmlTemplates *preloadedTemplates) {
	h.logger.Debug("connectToClient called", slog.Any("client", client))

	client.SendMessage(HtmlTemplates.ConnectionStatusSearching.Bytes())
	client.SendMessage(HtmlTemplates.ActionBtnSearching.Bytes())

	h.logger.Debug("finding matching client...")

	partner := h.OnlineClients.FindMatchingClient(client.SessionID, SEARCH_RETRIES, SEARCH_RETRY_WAIT)

	if partner == nil {
		h.logger.Debug("matching client not found")

		client.SendMessage(HtmlTemplates.ConnectionStatusNoClientsFound.Bytes())
		client.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
		return
	} else {
		h.logger.Debug("matching client found, attempting to connect")
		err := client.Connect(partner)
		if err != nil {
			h.logger.Debug("PARTNER: ", slog.Any("partner", partner))
			h.logger.Debug("matching client found, could not connect to found client")
			client.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
			return
		}

		client.SendMessage(HtmlTemplates.ConnectionStatusConnected.Bytes())
		partner.SendMessage(HtmlTemplates.ConnectionStatusConnected.Bytes())
		client.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
		partner.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())

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
}

func (h *Hub) handleRegistration(client *Client, HtmlTemplates *preloadedTemplates) {
	h.logger.Debug("handleRegistration called", slog.Any("client", client))
	defer h.wg.Done()

	if client == nil {
		return
	}

	h.connectToClient(client, HtmlTemplates)
}

func (h *Hub) handleUnregistration(client *Client, HtmlTemplates *preloadedTemplates) {
	h.logger.Debug("handleUnregistration called", slog.Any("client", client))
	defer h.wg.Done()

	if client == nil {
		return
	}

	client.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())

	partner := client.Disconnect()

	if partner != nil {
		partner.Disconnect()
		partner.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
	}
}

func (h *Hub) handleMessage(message *Message, HtmlTemplates *preloadedTemplates) {
	defer h.wg.Done()

	message_type := message.Type
	switch message_type {
	case "chat_message":
		h.handleChatMessage(message)
	case "typing":
		h.handleTypingMessage(message, HtmlTemplates)
	case "end_connection":
		h.handleEndCallMessage(message, HtmlTemplates)
	case "new_connection":
		h.handleNewCallMessage(message, HtmlTemplates)
	}
}

func (h *Hub) handleChatMessage(message *Message) {
	h.logger.Debug("handleChatMessage called", slog.Any("message", message))
	client := message.From
	if client == nil {
		return
	}

	partner := message.From.ChatPartner
	if partner == nil {
		return
	}

	clientMessageComponent := templates.ChatBubble(message.ChatMessage, true)
	partnerMessageComponent := templates.ChatBubble(message.ChatMessage, false)

	html := bytes.NewBuffer(nil)
	clientMessageComponent.Render(context.Background(), html)
	client.SendMessage(html.Bytes())

	html = bytes.NewBuffer(nil)
	partnerMessageComponent.Render(context.Background(), html)
	partner.SendMessage(html.Bytes())
}

func (h *Hub) handleTypingMessage(message *Message, HtmlTemplates *preloadedTemplates) {
	h.logger.Debug("handleTypingMessage called", slog.Any("message", message))
	client := message.From
	if client == nil {
		return
	}

	partner := message.From.ChatPartner
	if partner == nil {
		return
	}
	partner.SendMessage(HtmlTemplates.StrangerTyping.Bytes())

}

func (h *Hub) handleNewCallMessage(message *Message, HtmlTemplates *preloadedTemplates) {
	h.logger.Debug("handleNewCallMessage called", slog.Any("message", message))
	client := message.From
	if client == nil {
		return
	}

	// Disconnect
	partner := client.Disconnect()
	if partner != nil {
		client.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
		partner.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
	}

	h.connectToClient(client, HtmlTemplates)
}

func (h *Hub) handleEndCallMessage(message *Message, HtmlTemplates *preloadedTemplates) {
	h.logger.Debug("handleEndCallMessage called", slog.Any("message", message))
	client := message.From
	if client == nil {
		return
	}

	// Disconnect
	client.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())

	partner := client.Disconnect()
	if partner != nil {
		partner.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
	}
}

type preloadedTemplates struct {
	ConnectionStatusReady          *bytes.Buffer
	ConnectionStatusSearching      *bytes.Buffer
	ConnectionStatusConnected      *bytes.Buffer
	ConnectionStatusDisconnected   *bytes.Buffer
	ConnectionStatusNoClientsFound *bytes.Buffer
	StrangerTyping                 *bytes.Buffer
	ActionBtnNew                   *bytes.Buffer
	ActionBtnSearching             *bytes.Buffer
	ClientChatBubble               *bytes.Buffer
	PartnerChatBubble              *bytes.Buffer
}

func renderTemplate(templateFunc templ.Component) *bytes.Buffer {
	buffer := bytes.NewBuffer(nil)
	templateFunc.Render(context.Background(), buffer)
	return buffer
}

func preloadTemplates() *preloadedTemplates {
	return &preloadedTemplates{
		ConnectionStatusReady:          renderTemplate(templates.ConnectionStatusReady()),
		ConnectionStatusSearching:      renderTemplate(templates.ConnectionStatusSearching()),
		ConnectionStatusConnected:      renderTemplate(templates.ConnectionStatusConnected()),
		ConnectionStatusDisconnected:   renderTemplate(templates.ConnectionStatusDisconnected()),
		ConnectionStatusNoClientsFound: renderTemplate(templates.ConnectionStatusNoClientsFound()),
		StrangerTyping:                 renderTemplate(templates.StrangerTyping()),
		ActionBtnNew:                   renderTemplate(templates.ActionButton_NewChat()),
		ActionBtnSearching:             renderTemplate(templates.ActionButton_Searching()),
	}
}
