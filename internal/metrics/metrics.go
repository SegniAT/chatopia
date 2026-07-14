package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ---- Counters ----
var (
	VisitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "visits_total",
		Help: "Total page visits",
	}, []string{"page"})

	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests by status code, method, and route path",
	}, []string{"status", "method", "path"})

	SessionsCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sessions_created_total",
		Help: "Total chat sessions created",
	})

	ChatsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "chats_total",
		Help: "Total chats started",
	}, []string{"type"})

	MatchmakingAttemptsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matchmaking_attempts_total",
		Help: "Total matchmaking attempts",
	})

	MatchmakingFoundTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "matchmaking_found_total",
		Help: "Matchmaking results by found state and reason",
	}, []string{"found", "reason"})

	MatchmakingTimeoutTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matchmaking_timeout_total",
		Help: "Total matchmaking timeouts",
	})

	WSMessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ws_messages_total",
		Help: "WebSocket messages sent and received",
	}, []string{"direction"})
)

// ---- Gauges ----
var (
	ActiveClients = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "active_clients",
		Help: "Currently registered clients",
	})

	ActiveChats = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "active_chats",
		Help: "Currently active chats",
	}, []string{"type"})

	QueueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queue_depth",
		Help: "Current matchmaking queue depth",
	}, []string{"type"})
)

// ---- Histograms ----
var (
	HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	ChatDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "chat_duration_seconds",
		Help:    "Chat duration in seconds",
		Buckets: []float64{10, 30, 60, 120, 300, 600, 1800, 3600},
	}, []string{"type"})

	MatchmakingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "matchmaking_duration_seconds",
		Help:    "Time to find a match in seconds",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20, 30},
	})

	WSConnectionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ws_connection_duration_seconds",
		Help:    "WebSocket connection duration in seconds",
		Buckets: []float64{10, 30, 60, 120, 300, 600, 1800, 3600},
	})
)
