package websocket

import (
	"sync"
	"time"
)

type OnlineClients struct {
	clients sync.Map
}

func NewOnlineClientsStore() *OnlineClients {
	return &OnlineClients{}
}

func (activeClients *OnlineClients) Size() int {
	var count int
	activeClients.clients.Range(func(key, value any) bool {
		count++
		return true
	})

	return count
}

func (activeClients *OnlineClients) StoreClient(sessionID string, client *Client) {
	activeClients.clients.Store(sessionID, client)
}

func (activeClients *OnlineClients) GetClient(sessionID string) (*Client, bool) {
	client, exists := activeClients.clients.Load(sessionID)
	if !exists {
		return nil, false
	}

	return client.(*Client), true
}

func (activeClients *OnlineClients) DeleteClient(sessionID string) {
	activeClients.clients.Delete(sessionID)
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

	activeClients.clients.Range(func(_, value interface{}) bool {
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
	activeClients.clients.Range(func(_, value interface{}) bool {
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

func hasCommonInterests(interests1, interests2 []string) bool {
	for interest1 := range interests1 {
		for interest2 := range interests2 {
			if interest1 == interest2 {
				return true
			}
		}
	}
	return false
}

func countCommonInterests(interests1, interests2 []string) int {
	commonCount := 0
	for interest1 := range interests1 {
		for interest2 := range interests2 {
			if interest1 == interest2 {
				commonCount++
			}
		}
	}
	return commonCount
}
