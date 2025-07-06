package websocket

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/SegniAdebaGodsSon/ui/templates"
	"github.com/a-h/templ"
	"github.com/google/uuid"
)

var HtmlTemplates *preloadedTemplates

func init() {
	HtmlTemplates = preloadTemplates()
}

type Hub struct {
	Matchmaker *Matchmaker
	register   chan *Client
	unregister chan *Client
	Recieve    chan *Message
	ctx        context.Context
	wg         sync.WaitGroup
	logger     *slog.Logger
}

func NewHub(ctx context.Context, logger *slog.Logger) *Hub {
	return &Hub{
		Matchmaker: NewMatchmaker(logger),
		Recieve:    make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		ctx:        ctx,
		logger:     logger,
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

func (h *Hub) Run() {
	defer func() {
		h.Matchmaker.Stop()
		if r := recover(); r != nil {
			h.logger.ErrorContext(context.Background(), "panic in Hub.Run", slog.Any("error", r))
		}
	}()

	h.Matchmaker.onPair = h.handleMatchedPair
	h.Matchmaker.onTimeout = h.handleMatchTimeout

	h.logger.InfoContext(context.Background(), "Hub has started running")

	for {
		select {
		case <-h.ctx.Done():
			h.Matchmaker.Stop()
			h.wg.Wait()
			return

		case client := <-h.register:
			h.wg.Add(1)
			go h.handleRegistration(client)

		case client := <-h.unregister:
			h.wg.Add(1)
			go h.handleUnregistration(client)

		case message := <-h.Recieve:
			h.wg.Add(1)
			go h.handleMessage(message)
		}
	}
}

func (h *Hub) handleRegistration(client *Client) {
	h.logger.Debug("handleRegistration called", slog.Any("client", client))

	defer func() {
		if r := recover(); r != nil {
			h.logger.ErrorContext(context.Background(), "panic in Hub.handleRegistration", slog.Any("error", r))
		}
	}()

	defer h.wg.Done()

	client.SendMessage(HtmlTemplates.ConnectionStatusSearching.Bytes())
	client.SendMessage(HtmlTemplates.ActionBtnSearching.Bytes())

	h.Matchmaker.Submit(client)
}

func (h *Hub) handleUnregistration(client *Client) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.ErrorContext(context.Background(), "panic in Hub.handleUnregistration", slog.Any("error", r))
		}
	}()

	h.logger.Debug("handleUnregistration called", slog.Any("client", client))
	defer h.wg.Done()

	if client == nil {
		return
	}

	h.Matchmaker.RemoveClient(client.SessionID) // Queues
	h.Matchmaker.DeleteClient(client.SessionID) // General all client store
	h.logger.InfoContext(h.ctx,
		"client deleted from store",
		slog.Any("client", client),
	)

	renderAndSend := func(c *Client, peerID, strangerPeerID uuid.UUID, isClient bool) {
		buf := bytes.NewBuffer(nil)
		templates.ChatInner(
			peerID,
			strangerPeerID,
			isClient,
			c.ChatType == "video",
			c.Interests,
		).Render(h.ctx, buf)

		c.SendMessage(buf.Bytes())
	}

	renderAndSend(client, uuid.Nil, uuid.Nil, true)
	client.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())

	partner := client.Disconnect()
	if partner != nil {
		partner.Disconnect()
		renderAndSend(partner, uuid.Nil, uuid.Nil, false)
		partner.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
	}
}

func (h *Hub) handleMatchedPair(c1, c2 *Client) {
	c1.Connect(c2)

	// Generate peer IDs
	peer1 := uuid.New()
	peer2 := uuid.New()

	renderAndSend := func(c *Client, peerID, strangerPeerID uuid.UUID, isClient bool) {
		buf := bytes.NewBuffer(nil)
		templates.ChatInner(
			peerID,
			strangerPeerID,
			isClient,
			c.ChatType == "video",
			c.Interests,
		).Render(h.ctx, buf)

		c.SendMessage(buf.Bytes())
	}

	renderAndSend(c1, peer1, peer2, true)
	renderAndSend(c2, peer2, peer1, false)

	// Send status updates
	c1.SendMessage(HtmlTemplates.ConnectionStatusConnected.Bytes())
	c2.SendMessage(HtmlTemplates.ConnectionStatusConnected.Bytes())
	c1.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
	c2.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
}

func (h *Hub) handleMatchTimeout(client *Client) {
	client.SendMessage(HtmlTemplates.ConnectionStatusNoClientsFound.Bytes())
	client.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
}

func (h *Hub) handleMessage(message *Message) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.ErrorContext(context.Background(), "panic in Hub.handleMessage", slog.Any("error", r))
		}
	}()

	defer h.wg.Done()

	message_type := message.Type
	switch message_type {
	case "chat_message":
		h.handleChatMessage(message)
	case "typing":
		h.handleTypingMessage(message)
	case "end_connection":
		h.handleEndCallMessage(message)
	case "new_connection":
		h.handleNewCallMessage(message)
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

func (h *Hub) handleTypingMessage(message *Message) {
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

func (h *Hub) handleNewCallMessage(message *Message) {
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

	client.SendMessage(HtmlTemplates.ConnectionStatusSearching.Bytes())
	client.SendMessage(HtmlTemplates.ActionBtnSearching.Bytes())

	h.Matchmaker.Submit(client)
}

func (h *Hub) handleEndCallMessage(message *Message) {
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
