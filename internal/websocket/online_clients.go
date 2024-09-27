package websocket

import (
	"sync"
	"time"
)

type OnlineClients struct {
	sync.Map
}

func NewOnlineClientsStore() *OnlineClients {
	return &OnlineClients{}
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
	for range retries {
		partner := activeClients.findMatchingClientInternal(sessionID)
		if partner != nil {
			return partner
		}

		time.Sleep(waitDuration)
	}

	return nil
}

func (activeClients *OnlineClients) findMatchingClientInternal(sessionID string) *Client {
	currentClient, exists := activeClients.GetClient(sessionID)
	if !exists {
		return nil
	}
	currentClient.Searching = true
	defer func() {
		currentClient.Searching = false
	}()

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

		if client.Searching {
			return true
		}

		commonInterests := countCommonInterests(client.Interests, currentClient.Interests)
		if commonInterests > maxCommonInterests {
			bestMatch = client
			maxCommonInterests = commonInterests

			if maxCommonInterests == 3 {
				return false
			}
		}

		return true
	})

	// check if both clients are still available
	if bestMatch != nil && currentClient.ChatPartner == nil && bestMatch.ChatPartner == nil {
		return bestMatch
	}

	var anyClient *Client
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

		if client.Searching {
			return true
		}

		anyClient = client
		return false
	})

	if anyClient != nil && currentClient.ChatPartner == nil && anyClient.ChatPartner == nil {
		return anyClient
	}

	return nil
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
