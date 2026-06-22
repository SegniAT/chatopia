package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type MatchNotification struct {
	RoomID    string    `json:"room_id"`
	PartnerID string    `json:"partner_id"`
	ChatType  string    `json:"chat_type"`
	Timestamp time.Time `json:"timestamp"`
}

type Notifier struct {
	client *Client
}

func NewNotifier(c *Client) *Notifier {
	return &Notifier{client: c}
}

func (n *Notifier) NotifyMatch(ctx context.Context, session *Session) error {
	notify := MatchNotification{
		RoomID:    session.RoomID,
		PartnerID: session.PartnerID,
		ChatType:  session.ChatType,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(notify)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	channel := NotifyKey(session.SessionID)
	return n.client.Client.Publish(ctx, channel, data).Err()
}

func (n *Notifier) SubscribeMatch(ctx context.Context, sessionID string) (<-chan MatchNotification, func()) {
	channel := NotifyKey(sessionID)
	pubsub := n.client.Client.Subscribe(ctx, channel)

	notifyChan := make(chan MatchNotification, 1)
	closed := make(chan struct{})
	msgChan := pubsub.Channel()

	go func() {
		defer close(notifyChan)
		for {
			select {
			case <-ctx.Done():
				_ = pubsub.Close()
				return
			case <-closed:
				_ = pubsub.Close()
				return
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
				var notify MatchNotification
				if err := json.Unmarshal([]byte(msg.Payload), &notify); err != nil {
					continue
				}
				notifyChan <- notify
				return
			}
		}
	}()

	cleanup := func() {
		select {
		case <-closed:
		default:
			close(closed)
		}
	}

	return notifyChan, cleanup
}
