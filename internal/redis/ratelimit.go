package redis

import (
	"context"
	"fmt"
	"time"
)

func (c *Client) AllowIP(ctx context.Context, ip string, maxBurst int) (bool, error) {
	key := "rl:" + ip

	count, err := c.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("rate limit incr failed: %w", err)
	}

	if count == 1 {
		if err := c.Expire(ctx, key, time.Second).Err(); err != nil {
			return false, fmt.Errorf("rate limit expire failed: %w", err)
		}
	}

	return count <= int64(maxBurst), nil
}
