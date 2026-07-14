package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/SegniAT/internal/metrics"
	"github.com/SegniAT/internal/redis"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("statusWriter does not wrap an http.Hijacker")
}

// metricsMiddleware records HTTP request count and duration.
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		routePath := r.Pattern
		if routePath == "" {
			routePath = "unknown"
		}

		// Record duration
		metrics.HTTPDuration.WithLabelValues(r.Method, routePath).Observe(time.Since(start).Seconds())

		// Record requests total with all 3 dimensional labels
		metrics.HTTPRequestsTotal.WithLabelValues(
			strconv.Itoa(sw.status),
			r.Method,
			routePath,
		).Inc()
	})
}

// metricsHandler exposes the /metrics endpoint for Prometheus scraping.
func metricsHandler() http.Handler {
	return promhttp.Handler()
}

// startQueueDepthPoller reads queue sizes from Redis every 5 seconds.
func startQueueDepthPoller(ctx context.Context, rdb *redis.Client) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, t := range []string{"text", "video"} {
				val, err := rdb.ZCard(ctx, redis.QueueKey(t)).Result()
				if err != nil {
					continue
				}
				metrics.QueueDepth.WithLabelValues(t).Set(float64(val))
			}
		}
	}
}
