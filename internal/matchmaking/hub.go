package matchmaking

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/SegniAT/internal/metrics"
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
	clients sync.Map

	clientCount atomic.Int64

	RedisClient *redis.Client
	Notifier    *redis.Notifier

	// ctx/cancel controls the lifecycle of goroutines owned by Hub
	// (Start loop, ReadPump, WritePump, waitForMatch).
	// Redis operations use context.Background() so cleanup completes during shutdown.
	ctx    context.Context
	cancel context.CancelFunc

	handlerWg sync.WaitGroup
	clientWg  sync.WaitGroup

	receive chan *Message

	TurnUser       string
	TurnCredential string
}

func NewHub(ctx context.Context, redisClient *redis.Client, turnUser string, turnCredential string) *Hub {
	ctx, cancel := context.WithCancel(ctx)
	return &Hub{
		RedisClient:    redisClient,
		Notifier:       redis.NewNotifier(redisClient),
		ctx:            ctx,
		cancel:         cancel,
		receive:        make(chan *Message, 256),
		TurnUser:       turnUser,
		TurnCredential: turnCredential,
	}
}

func (h *Hub) GetClient(sessionID string) (*Client, bool) {
	val, ok := h.clients.Load(sessionID)
	if !ok {
		return nil, false
	}

	return val.(*Client), true
}

func (h *Hub) RegisterClient(client *Client) error {
	if client == nil {
		return fmt.Errorf("can't register nil client")
	}

	if h.RedisClient != nil {
		session := redis.NewSession(client.SessionID, client.ChatType, client.IsStrict, client.Interests, time.Now())
		if err := session.Save(context.Background(), h.RedisClient); err != nil {
			return fmt.Errorf("Hub.RegisterClient error saving session: %v", err)
		}
	}

	h.clients.Store(client.SessionID, client)
	metrics.SessionsCreatedTotal.Inc()

	slog.Info("client registered",
		slog.String("session_id", client.SessionID),
		slog.String("chat_type", client.ChatType),
	)

	return nil
}

func (h *Hub) UnregisterClient(client *Client) error {
	if client == nil {
		return fmt.Errorf("can't unregister nil client")
	}

	h.clients.Delete(client.SessionID)
	metrics.ActiveClients.Dec()

	// TODO: this probably won't work since the connection is probably closed already
	client.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
	client.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())

	partner := client.Disconnect()
	if partner != nil {
		partner.ChatPartner = nil
		partner.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
		partner.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
		metrics.ActiveChats.WithLabelValues(client.ChatType).Dec()
		if !client.ChatStartedAt.IsZero() {
			metrics.ChatDuration.WithLabelValues(client.ChatType).Observe(float64(time.Since(client.ChatStartedAt).Seconds()))
			client.ChatStartedAt = time.Time{}
		}
	}

	if !client.ConnStartedAt.IsZero() {
		metrics.WSConnectionDuration.Observe(float64(time.Since(client.ConnStartedAt).Seconds()))
	}

	slog.Info("client unregistered",
		slog.String("session_id", client.SessionID),
	)

	h.clientCount.Add(-1)

	if h.RedisClient != nil {
		if err := redis.RemoveFromQueue(context.Background(), h.RedisClient, client.ChatType, client.SessionID); err != nil {
			slog.Error("failed to remove client from queue",
				slog.String("session_id", client.SessionID),
				slog.String("error", err.Error()),
			)
		}

		if err := redis.DeleteSession(context.Background(), h.RedisClient, client.SessionID); err != nil {
			slog.Error("failed to delete session",
				slog.String("session_id", client.SessionID),
				slog.String("error", err.Error()),
			)
		}
	}

	return nil
}

func (h *Hub) ClientCount() int64 {
	return h.clientCount.Load()
}

func (h *Hub) Start() {
	go func() {
		slog.Info("Hub started")
		defer slog.Info("Hub stopped")

		for {
			select {
			case <-h.ctx.Done():
				h.handlerWg.Wait()
				return

			case message := <-h.receive:
				h.handlerWg.Go(func() {
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

						metrics.ActiveChats.WithLabelValues(message.From.ChatType).Inc()
						message.From.ChatStartedAt = time.Now()

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
								metrics.ActiveChats.WithLabelValues(message.From.ChatType).Dec()
								if !message.From.ChatStartedAt.IsZero() {
									metrics.ChatDuration.WithLabelValues(message.From.ChatType).Observe(float64(time.Since(message.From.ChatStartedAt).Seconds()))
									message.From.ChatStartedAt = time.Time{}
								}
							}
						}

						if h.RedisClient != nil {
							redis.ClearPartnerFields(context.Background(), h.RedisClient, message.From.SessionID)
							if partnerSessionID != "" {
								redis.ClearPartnerFields(context.Background(), h.RedisClient, partnerSessionID)
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
								metrics.ActiveChats.WithLabelValues(message.From.ChatType).Dec()
								if !message.From.ChatStartedAt.IsZero() {
									metrics.ChatDuration.WithLabelValues(message.From.ChatType).Observe(float64(time.Since(message.From.ChatStartedAt).Seconds()))
									message.From.ChatStartedAt = time.Time{}
								}
							}
						}

						message.From.SendMessage(HtmlTemplates.ConnectionStatusDisconnected.Bytes())
						message.From.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
					}
				})
			}
		}
	}()
}

func (h *Hub) Stop() {
	// Only for good measure, the connections are already closed by the client ReadPump and WritePump goroutines when the hub is stopped.
	h.clients.Range(func(_, value any) bool {
		client := value.(*Client)
		if client.Conn != nil {
			client.Conn.Close()
		}
		return true
	})

	h.cancel()
	h.clientWg.Wait()
	h.handlerWg.Wait()
}

func (h *Hub) ClientConnected(client *Client) {
	h.clientCount.Add(1)
	metrics.ActiveClients.Inc()
	h.clientWg.Go(func() {
		client.WritePump()
	})
	h.clientWg.Go(func() {
		client.ReadPump()
	})
}

func (h *Hub) StartMatchmaking(client *Client) {
	metrics.MatchmakingAttemptsTotal.Inc()

	go func() {
		start := time.Now()
		defer func() {
			metrics.MatchmakingDuration.Observe(float64(time.Since(start).Seconds()))

			if r := recover(); r != nil {
				slog.Error("panic in Hub.StartMatchmaking", slog.Any("error", r))
			}
		}()

		session, err := redis.GetSession(context.Background(), h.RedisClient, client.SessionID)
		if err != nil {
			slog.Error("failed to get session for matchmaking", slog.String("error", err.Error()))
			return
		}

		result, err := redis.Match(context.Background(), h.RedisClient, session)
		if err != nil {
			slog.Error("matchmaking failed", slog.String("error", err.Error()))
			client.SendMessage(HtmlTemplates.ConnectionStatusNoClientsFound.Bytes())
			client.SendMessage(HtmlTemplates.ActionBtnNew.Bytes())
			return
		}

		if result.Found {
			metrics.MatchmakingFoundTotal.WithLabelValues("true", result.Reason).Inc()

			partner, err := redis.GetSession(context.Background(), h.RedisClient, result.Partner.SessionID)
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
	notifyChan, cleanup := h.Notifier.SubscribeMatch(context.Background(), client.SessionID)
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

				partner, err := redis.GetSession(context.Background(), h.RedisClient, notify.PartnerID)
				if err != nil {
					slog.Error("failed to get partner session", slog.String("error", err.Error()))
					return
				}

				h.connectPairedClients(client, partner)
				return
			}
		case <-timeout:
			metrics.MatchmakingTimeoutTotal.Inc()
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

	// Notify the partner first — if this fails, no state has been mutated
	if err := h.Notifier.NotifyMatch(context.Background(), partner); err != nil {
		slog.Error("failed to notify partner of match, aborting connection",
			slog.String("error", err.Error()),
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
		templates.ChatInner(peerID, strangerPeerID, isCaller, c.ChatType == "video", h.TurnUser, h.TurnCredential).
			Render(h.ctx, buf)
		c.SendMessage(buf.Bytes())
	}

	renderAndSend(c1, peer1, peer2, true)
	renderAndSend(c2, peer2, peer1, false)

	c1.SendMessage(HtmlTemplates.ConnectionStatusMatchFoundConnecting.Bytes())
	c1.SendMessage(HtmlTemplates.ActionBtnConnecting.Bytes())
	c2.SendMessage(HtmlTemplates.ConnectionStatusMatchFoundConnecting.Bytes())
	c2.SendMessage(HtmlTemplates.ActionBtnConnecting.Bytes())

	slog.Info("clients matched",
		slog.String("client1", c1.SessionID),
		slog.String("client2", c2.SessionID),
	)

	metrics.ChatsTotal.WithLabelValues(c1.ChatType).Inc()
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
