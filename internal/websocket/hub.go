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
	register      chan *Client
	unregister    chan *Client
	Recieve       chan *Message
	ctx           context.Context
	wg            sync.WaitGroup
	logger        *slog.Logger
}

func NewHub(ctx context.Context, logger *slog.Logger) *Hub {
	return &Hub{
		OnlineClients: NewOnlineClientsStore(logger),
		Recieve:       make(chan *Message),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		ctx:           ctx,
		logger:        logger,
	}
}

func (h *Hub) RegisterClient(client *Client) error {
	if client == nil {
		return fmt.Errorf("can't register nil 'Client'")
	}
	h.register <- client
	return nil
}

func (h *Hub) UnregisterClient(client *Client) error {
	if client == nil {
		return fmt.Errorf("can't unregister nil 'Client'")
	}
	h.unregister <- client
	return nil
}

/*
func (h *Hub) RunMatchmaker(HtmlTemplates *preloadedTemplates) {
	for {
		select {
		case client := <-h.queue:
			h.connectToClient(client, HtmlTemplates)
		}
	}
}
*/

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

		case client := <-h.register:
			h.wg.Add(1)
			go h.handleRegistration(client, HtmlTemplates)

		case client := <-h.unregister:
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

	if client.ChatPartner != nil {
		h.logger.Debug("matching client already found")
		return
	}

	if partner == nil {
		h.logger.Debug("matching client not found")

		client.SendMessage(HtmlTemplates.ConnectionStatusNoClientsFound.Bytes())
		client.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
		return
	}

	h.logger.Debug("matching client found, attempting to connect")
	err := client.Connect(partner)
	if err != nil {
		h.logger.Debug("PARTNER: ", slog.Any("partner", partner))
		h.logger.Debug("matching client found, could not connect to found client", slog.String("error", err.Error()))
		client.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
		return
	}

	chatInnerClientBytesBuffer := bytes.NewBuffer(nil)
	chatInnerStrangerBytesBuffer := bytes.NewBuffer(nil)
	var peerID = uuid.New()
	var strangerPeerID = uuid.New()

	templates.ChatInner(
		peerID,
		strangerPeerID,
		true,
		client.ChatType == "video",
		client.Interests,
	).Render(h.ctx, chatInnerClientBytesBuffer)

	templates.ChatInner(
		strangerPeerID,
		peerID,
		false,
		partner.ChatType == "video",
		partner.Interests,
	).Render(h.ctx, chatInnerStrangerBytesBuffer)

	client.SendMessage(chatInnerClientBytesBuffer.Bytes())
	partner.SendMessage(chatInnerStrangerBytesBuffer.Bytes())

	client.SendMessage(HtmlTemplates.ConnectionStatusConnected.Bytes())
	partner.SendMessage(HtmlTemplates.ConnectionStatusConnected.Bytes())
	client.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
	partner.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
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

	partner := client.Disconnect()
	client.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())

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

	partner := client.Disconnect()
	client.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())

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
