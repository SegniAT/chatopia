package matchmaking

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/SegniAT/internal/redis"
	"github.com/SegniAT/ui/templates"
	"github.com/a-h/templ"
	"github.com/google/uuid"
)

var HtmlTemplates preloadedTemplates

func init() {
	HtmlTemplates = preloadTemplates()
}

const matchTimeout = 30 * time.Second

type Hub struct {
	// clients stores online clients.
	clients     sync.Map
	RedisClient *redis.Client
	Notifier    *redis.Notifier
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	Receive     chan *Message
}

func NewHub(ctx context.Context, redisClient *redis.Client) *Hub {
	ctx, cancel := context.WithCancel(ctx)
	return &Hub{
		RedisClient: redisClient,
		Notifier:    redis.NewNotifier(redisClient),
		ctx:         ctx,
		cancel:      cancel,
		Receive:     make(chan *Message, 256),
	}
}

func (h *Hub) RegisterClient(client *Client) error {
	if client == nil {
		return fmt.Errorf("can't register nil client")
	}

	h.clients.Store(client.SessionID, client)

	if h.RedisClient != nil {
		session := redis.NewSession(client.SessionID, client.ChatType, client.IsStrict, client.Interests, time.Now())
		err := session.Save(h.ctx, h.RedisClient)
		if err != nil {
			return fmt.Errorf("Hub.RegisterClient error saving session: %v", err)
		}
	}

	slog.Info("client registered",
		slog.String("session_id", client.SessionID),
		slog.String("chat_type", client.ChatType),
	)

	return nil
}

func (h *Hub) GetClient(sessionID string) (*Client, bool) {
	val, ok := h.clients.Load(sessionID)
	if !ok {
		return nil, false
	}

	return val.(*Client), true
}

func (h *Hub) UnregisterClient(client *Client) error {
	if client == nil {
		return fmt.Errorf("can't unregister nil client")
	}

	if h.RedisClient != nil {
		redis.RemoveFromQueue(h.ctx, h.RedisClient, client.ChatType, client.SessionID)

		if err := redis.DeleteSession(h.ctx, h.RedisClient, client.SessionID); err != nil {
			return fmt.Errorf("Hub.UnregisterClient redis.DeleteSession: %w", err)
		}
	}

	h.clients.Delete(client.SessionID)

	client.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
	client.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())

	partner := client.ChatPartner
	client.ChatPartner = nil
	if partner != nil {
		partner.ChatPartner = nil
		partner.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
		partner.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
	}

	slog.Info("client unregistered",
		slog.String("session_id", client.SessionID),
	)

	return nil
}

func (h *Hub) ClientCount() int {
	count := 0
	h.clients.Range(func(_, _ any) bool {
		count++
		return true
	})

	return count
}

func (h *Hub) Start() {
	go func() {
		slog.Info("Hub started")
		defer slog.Info("Hub stopped")

		for {
			select {
			case <-h.ctx.Done():
				h.wg.Wait()
				return

			case message := <-h.Receive:
				h.wg.Add(1)
				go func() {
					defer h.wg.Done()

					switch message.Type {
					case "typing":
						if message.From == nil || message.From.ChatPartner == nil {
							slog.Warn("HUB.handleTyping client or partner are nil")
							return
						}

						message.From.ChatPartner.SendMessage(HtmlTemplates.StrangerTyping.Bytes())

					case "peer_connected":
						if message.From == nil {
							slog.Warn("HUB.handlePeerConnected client is nil")
							return
						}

						message.From.SendMessage(HtmlTemplates.ConnectionStatusConnected.Bytes())
						message.From.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())

						if message.From.ChatPartner == nil {
							slog.Warn("HUB.handlePeerConnected partner is nil")
							return
						}

						message.From.ChatPartner.SendMessage(HtmlTemplates.ConnectionStatusConnected.Bytes())
						message.From.ChatPartner.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())

					case "new_connection":
						if message.From == nil {
							slog.Warn("HUB.handleNewConnection client is nil")
							return
						}

						var partnerSessionID string
						if message.From.ChatPartner != nil {
							partner := message.From.Disconnect()
							if partner != nil {
								partnerSessionID = partner.SessionID
								partner.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
							}
						}

						if h.RedisClient != nil {
							redis.ClearPartnerFields(h.ctx, h.RedisClient, message.From.SessionID)
							if partnerSessionID != "" {
								redis.ClearPartnerFields(h.ctx, h.RedisClient, partnerSessionID)
							}
						}

						message.From.SendMessage(HtmlTemplates.ConnectionStatusSearching.Bytes())
						message.From.SendMessage(HtmlTemplates.ActionBtnSearching.Bytes())

						h.StartMatchmaking(message.From)

					case "end_connection":
						if message.From == nil {
							slog.Warn("HUB.handleEndConnection client is nil")
							return
						}

						if message.From.ChatPartner != nil {
							partner := message.From.Disconnect()
							if partner != nil {
								partner.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
							}
						}

						message.From.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
						message.From.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
					}
				}()
			}
		}
	}()
}

func (h *Hub) Stop() {
	h.cancel()
	h.Notifier.Close()
}

func (h *Hub) StartMatchmaking(client *Client) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in Hub.StartMatchmaking", slog.Any("error", r))
			}
		}()

		session, err := redis.GetSession(h.ctx, h.RedisClient, client.SessionID)
		if err != nil {
			slog.Error("failed to get session for matchmaking", slog.String("error", err.Error()))
			return
		}

		result, err := redis.Match(h.ctx, h.RedisClient, session)
		if err != nil {
			slog.Error("matchmaking failed", slog.String("error", err.Error()))
			client.SendMessage(HtmlTemplates.ConnectionStatusNoClientsFound.Bytes())
			client.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
			return
		}

		if result.Found {
			partner, err := redis.GetSession(h.ctx, h.RedisClient, result.Partner.SessionID)
			if err != nil {
				slog.Error("failed to get partner session", slog.String("error", err.Error()))
				return
			}

			h.connectPairedClients(client, partner)

		} else {
			h.waitForMatch(client)
		}
	}()
}

func (h *Hub) waitForMatch(client *Client) {
	notifyChan, cleanup := h.Notifier.SubscribeMatch(h.ctx, client.SessionID)
	defer cleanup()

	timeout := time.After(matchTimeout)

	for {
		select {
		case <-h.ctx.Done():
			return
		case notify := <-notifyChan:
			if notify.RoomID != "" {
				// The matcher's goroutine may have already connected both clients
				if client.ChatPartner != nil {
					return
				}

				partner, err := redis.GetSession(h.ctx, h.RedisClient, notify.PartnerID)
				if err != nil {
					slog.Error("failed to get partner session", slog.String("error", err.Error()))
					return
				}

				h.connectPairedClients(client, partner)
				return
			}
		case <-timeout:
			slog.Info("matchmaking timeout",
				slog.String("session_id", client.SessionID),
			)
			client.SendMessage(HtmlTemplates.ConnectionStatusNoClientsFound.Bytes())
			client.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
			return
		}
	}
}

func (h *Hub) connectPairedClients(c1 *Client, partner *redis.Session) {
	c2, ok := h.GetClient(partner.SessionID)
	if !ok {
		slog.Warn("partner not connected via WebSocket, skipping UI render",
			slog.String("partner_id", partner.SessionID),
		)
		return
	}

	if err := c1.Connect(c2); err != nil {
		slog.Error("failed to connect clients", slog.String("error", err.Error()))
		return
	}

	peer1 := uuid.New()
	peer2 := uuid.New()

	renderAndSend := func(c *Client, peerID, strangerPeerID uuid.UUID, isCaller bool) {
		buf := bytes.NewBuffer(nil)
		templates.ChatInner(peerID, strangerPeerID, isCaller, c.ChatType == "video").
			Render(h.ctx, buf)
		c.SendMessage(buf.Bytes())
	}

	renderAndSend(c1, peer1, peer2, true)
	renderAndSend(c2, peer2, peer1, false)

	c1.SendMessage(HtmlTemplates.ConnectionStatusMatchFoundConnecting.Bytes())
	c1.SendMessage(HtmlTemplates.ActionBtnConnecting.Bytes())
	c2.SendMessage(HtmlTemplates.ConnectionStatusMatchFoundConnecting.Bytes())
	c2.SendMessage(HtmlTemplates.ActionBtnConnecting.Bytes())

	// Notify the partner in case they're waiting in waitForMatch
	if err := h.Notifier.NotifyMatch(h.ctx, partner); err != nil {
		slog.Error("failed to notify partner of match", slog.String("error", err.Error()))
	}

	slog.Info("clients matched",
		slog.String("client1", c1.SessionID),
		slog.String("client2", c2.SessionID),
	)
}

type preloadedTemplates struct {
	ConnectionStatusReady                *bytes.Buffer
	ConnectionStatusSearching            *bytes.Buffer
	ConnectionStatusMatchFoundConnecting *bytes.Buffer
	ConnectionStatusConnected            *bytes.Buffer
	ConnectionStatusDisconnected         *bytes.Buffer
	ConnectionStatusNoClientsFound       *bytes.Buffer
	StrangerTyping                       *bytes.Buffer
	ActionBtnNew                         *bytes.Buffer
	ActionBtnSearching                   *bytes.Buffer
	ActionBtnConnecting                  *bytes.Buffer
	ClientChatBubble                     *bytes.Buffer
	PartnerChatBubble                    *bytes.Buffer
}

func renderTemplate(templateFunc templ.Component) *bytes.Buffer {
	buf := bytes.NewBuffer(nil)
	_ = templateFunc.Render(context.Background(), buf)
	return buf
}

func preloadTemplates() preloadedTemplates {
	return preloadedTemplates{
		ConnectionStatusReady:                renderTemplate(templates.ConnectionStatusReady()),
		ConnectionStatusSearching:            renderTemplate(templates.ConnectionStatusSearching()),
		ConnectionStatusMatchFoundConnecting: renderTemplate(templates.ConnectionStatusMatchFoundConnecting()),
		ConnectionStatusConnected:            renderTemplate(templates.ConnectionStatusConnected()),
		ConnectionStatusDisconnected:         renderTemplate(templates.ConnectionStatusDisconnected()),
		ConnectionStatusNoClientsFound:       renderTemplate(templates.ConnectionStatusNoClientsFound()),
		StrangerTyping:                       renderTemplate(templates.StrangerTyping()),
		ActionBtnNew:                         renderTemplate(templates.ActionButton_NewChat()),
		ActionBtnSearching:                   renderTemplate(templates.ActionButton_Searching()),
		ActionBtnConnecting:                  renderTemplate(templates.ActionButton_Connecting()),
	}
}
