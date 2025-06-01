package websocket

import (
	"log/slog"
	"sync"
	"time"
)

type OnlineClients struct {
	logger *slog.Logger
	sync.Map
}

func NewOnlineClientsStore(logger *slog.Logger) *OnlineClients {
	return &OnlineClients{
		logger: logger,
	}
}

func (activeClients *OnlineClients) Size() int {
	var count int
	activeClients.Range(func(key, value any) bool {
		count++
		return true
	})

	return count
}

func (activeClients *OnlineClients) StoreClient(sessionID string, client *Client) {
	activeClients.Store(sessionID, client)
}

func (activeClients *OnlineClients) GetClient(sessionID string) (*Client, bool) {
	client, exists := activeClients.Load(sessionID)
	if !exists {
		return nil, false
	}

	return client.(*Client), true
}

func (activeClients *OnlineClients) DeleteClient(sessionID string) {
	activeClients.Delete(sessionID)
}

func (activeClients *OnlineClients) FindMatchingClient(
	sessionID string,
	retries int,
	waitDuration time.Duration,
) *Client {
	currentClient, exists := activeClients.GetClient(sessionID)
	if !exists {
		return nil
	}

	currentClient.Searching = true
	defer func() {
		currentClient.Searching = false
	}()

	for range retries {
		partner := activeClients.findMatchingClientInternal(currentClient)
		if partner != nil {
			return partner
		}

		time.Sleep(waitDuration)
	}

	return nil
}

func (activeClients *OnlineClients) findMatchingClientInternal(currentClient *Client) *Client {
	var bestMatch *Client
	maxCommonInterests := -1

	activeClients.Range(func(_, value interface{}) bool {
		client := value.(*Client)
		if !client.IsActive() {
			return true
		}

		if client.SessionID == currentClient.SessionID {
			return true
		}

		if client.ChatType != currentClient.ChatType {
			return true
		}

		if client.ChatPartner != nil || currentClient.ChatPartner != nil {
			return true
		}

		if !client.Searching {
			return true
		}

		commonInterests := countCommonInterests(client.Interests, currentClient.Interests)
		if len(currentClient.Interests) == 0 {
			bestMatch = client
			return false
		}

		if commonInterests > maxCommonInterests {
			bestMatch = client
			maxCommonInterests = commonInterests

			if maxCommonInterests == 3 {
				return false
			}
		}

		return true
	})

	if bestMatch == nil {
		return bestMatch
	}

	// if a client sets isStrict=true and has no interests, only match with people with no interests
	if (currentClient.IsStrict && len(currentClient.Interests) == 0) && (len(bestMatch.Interests) != 0) {
		return nil
	}

	if currentClient.IsStrict && maxCommonInterests == 0 ||
		bestMatch.IsStrict && maxCommonInterests == 0 {
		return nil
	}

	return bestMatch
}

func hasCommonInterests(ints1, ints2 []string) bool {
	for _, int1 := range ints1 {
		for _, int2 := range ints2 {
			if int1 == int2 {
				return true
			}
		}
	}
	return false
}

func countCommonInterests(ints1, ints2 []string) int {
	commonCount := 0
	for _, int1 := range ints1 {
		for _, int2 := range ints2 {
			if int1 == int2 {
				commonCount++
			}
		}
	}
	return commonCount
}
