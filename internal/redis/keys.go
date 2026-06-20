package redis

import "fmt"

const (
	KeyPrefixQueue   = "queue:"
	KeyPrefixSession = "session:"
	KeyPrefixNotify  = "notify:"
)

func QueueKey(chatType string) string {
	return fmt.Sprintf("%s%s", KeyPrefixQueue, chatType)
}

func SessionKey(sessionID string) string {
	return fmt.Sprintf("%s%s", KeyPrefixSession, sessionID)
}

func NotifyKey(sessionID string) string {
	return fmt.Sprintf("%s%s", KeyPrefixNotify, sessionID)
}
