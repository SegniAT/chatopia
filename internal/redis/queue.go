package redis

import (
	"context"
	"fmt"
)

func RemoveFromQueue(ctx context.Context, c *Client, chatType, sessionID string) error {
	key := QueueKey(chatType)
	if err := c.ZRem(ctx, key, sessionID).Err(); err != nil {
		return fmt.Errorf("failed to remove %s from queue %s: %w", sessionID, key, err)
	}
	return nil
}
