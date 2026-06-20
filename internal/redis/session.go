package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Session struct {
	SessionID string   `json:"session_id"`
	ChatType  string   `json:"chat_type"` // "text" | "video"
	IsStrict  bool     `json:"is_strict"`
	Interests []string `json:"interests"`
	QueuedAt  int64    `json:"queued_at"` // Unix timestamp
	PartnerID string   `json:"partner_id"`
	RoomID    string   `json:"room_id"`
	CreatedAt int64    `json:"created_at"`
}

func NewSession(sessionID, chatType string, isStrict bool, interests []string, now time.Time) *Session {
	return &Session{
		SessionID: sessionID,
		ChatType:  chatType,
		IsStrict:  isStrict,
		Interests: interests,
		QueuedAt:  now.Unix(),
		CreatedAt: now.Unix(),
	}
}

func (s *Session) Save(ctx context.Context, redisClient *Client) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := SessionKey(s.SessionID)
	if err := redisClient.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func GetSession(ctx context.Context, redisClient *Client, sessionID string) (*Session, error) {
	key := SessionKey(sessionID)
	data, err := redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func ClearPartnerFields(ctx context.Context, c *Client, sessionID string) error {
	session, err := GetSession(ctx, c, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session for partner clear: %w", err)
	}
	session.PartnerID = ""
	session.RoomID = ""
	return session.Save(ctx, c)
}

func DeleteSession(ctx context.Context, c *Client, sessionID string) error {
	key := SessionKey(sessionID)
	if err := c.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}
