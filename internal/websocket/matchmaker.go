package websocket

import (
	"container/list"
	"log/slog"
	"slices"
	"sync"
	"time"
)

const (
	SEARCH_TIMEOUT = 5 * time.Second
	WORKER_COUNT   = 16
	QUEUE_SIZE     = 10_000
)

type Matchmaker struct {
	logger *slog.Logger

	queues map[string]*MatchQueue

	timeouts sync.Map // map[string]*time.Timer

	workchan chan *Client
	stopchan chan struct{}

	onPair    func(c1, c2 *Client)
	onTimeout func(c *Client)
}

type MatchQueue struct {
	mu   sync.Mutex
	list *list.List
}

func NewMatchQueue() *MatchQueue {
	return &MatchQueue{
		list: list.New(),
	}
}

func NewMatchmaker(logger *slog.Logger) *Matchmaker {
	m := &Matchmaker{
		logger: logger,
		queues: map[string]*MatchQueue{
			"text":  NewMatchQueue(),
			"video": NewMatchQueue(),
		},
		workchan: make(chan *Client, QUEUE_SIZE),
		stopchan: make(chan struct{}),
	}

	for range WORKER_COUNT {
		go m.worker()
	}

	return m
}

func (m *Matchmaker) worker() {
	for {
		select {
		case c := <-m.workchan:
			m.processClient(c)
		case <-m.stopchan:
			return
		}
	}
}

func (m *Matchmaker) setTimeout(c *Client, d time.Duration) {
	timer := time.AfterFunc(d, func() {
		queue := m.queues[c.ChatType]
		queue.mu.Lock()
		defer queue.mu.Unlock()

		for e := queue.list.Front(); e != nil; e = e.Next() {
			if e.Value.(*Client).SessionID == c.SessionID {
				queue.list.Remove(e)
				break
			}
		}

		m.timeouts.Delete(c.SessionID)
		if m.onTimeout != nil {
			m.onTimeout(c)
		}
	})

	m.timeouts.Store(c.SessionID, timer)
}

func (m *Matchmaker) cancelTimeout(sessionID string) {
	if t, ok := m.timeouts.Load(sessionID); ok {
		t.(*time.Timer).Stop()
		m.timeouts.Delete(sessionID)
	}
}

func hasCommonInterest(i1, i2 []string) bool {
	for _, a := range i1 {
		if slices.Contains(i2, a) {
			return true
		}
	}

	return false
}

func (m *Matchmaker) isCompatible(c1, c2 *Client) bool {
	if c1.SessionID == c2.SessionID || c1.ChatType != c2.ChatType {
		return false
	}

	if c1.IsStrict || c2.IsStrict {
		return hasCommonInterest(c1.Interests, c2.Interests)
	}

	return true
}

func (m *Matchmaker) findMatchInQueue(queue *MatchQueue, client *Client) *Client {
	for e := queue.list.Front(); e != nil; e = e.Next() {
		candidate := e.Value.(*Client)
		if m.isCompatible(client, candidate) {
			queue.list.Remove(e)
			return candidate
		}
	}

	return nil
}

// NOTE: improve this locking mechanism in the future...I'm done for now!
func (m *Matchmaker) processClient(c *Client) {
	queue := m.queues[c.ChatType]
	queue.mu.Lock()
	defer queue.mu.Unlock()

	m.cancelTimeout(c.SessionID)

	match := m.findMatchInQueue(queue, c)
	if match != nil {
		m.cancelTimeout(match.SessionID)
		if m.onPair != nil {
			m.onPair(c, match)
		}
	} else {
		queue.list.PushBack(c)
		m.setTimeout(c, SEARCH_TIMEOUT)
	}
}

func (m *Matchmaker) Stop() {
	close(m.stopchan)
}

func (m *Matchmaker) Submit(c *Client) {
	select {
	case m.workchan <- c:
	default:
		m.logger.Warn("matchmaker queue full, client dropped")
		if m.onTimeout != nil {
			m.onTimeout(c)
		}
	}
}

func (m *Matchmaker) RemoveClient(sessionID string) {
	for _, queue := range m.queues {
		queue.mu.Lock()
		for e := queue.list.Front(); e != nil; e = e.Next() {
			if e.Value.(*Client).SessionID == sessionID {
				queue.list.Remove(e)
				break
			}
		}
		queue.mu.Unlock()
	}
	m.cancelTimeout(sessionID)
}

func (m *Matchmaker) ClientsCount() int {
	var size int
	for _, q := range m.queues {
		size += q.list.Len()
	}
	return size
}

var clientStore sync.Map // map[string]*Client

func (m *Matchmaker) AddClient(c *Client) {
	clientStore.Store(c.SessionID, c)
}

func (m *Matchmaker) GetClient(sessionID string) (*Client, bool) {
	val, ok := clientStore.Load(sessionID)
	if !ok {
		return nil, false
	}
	return val.(*Client), true
}
