package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/SegniAT/logger"
	"github.com/google/uuid"
)

// realIPMiddleware overrides r.RemoteAddr with the real client IP address
// extracted from trusted proxy headers (X-Forwarded-For, X-Real-IP).
// It must run as the first middleware so all downstream handlers and
// middleware — including logging, rate limiting, and WebSocket upgrade —
// read the correct client IP from r.RemoteAddr without additional calls.
func (app *application) realIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = func(r *http.Request) string {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				parts := strings.Split(xff, ",")
				return strings.TrimSpace(parts[0])
			}
			if xri := r.Header.Get("X-Real-IP"); xri != "" {
				return xri
			}
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				return r.RemoteAddr
			}
			return ip
		}(r)

		next.ServeHTTP(w, r)
	})
}

func (app *application) addRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := logger.AppendCtx(r.Context(), slog.String("req_id", uuid.NewString()))
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.InfoContext(r.Context(),
			"incoming request",
			slog.String("method", r.Method),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("uri", r.URL.RequestURI()),
			slog.String("proto", r.Proto),
		)
		next.ServeHTTP(w, r)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				slog.ErrorContext(r.Context(), "panic", slog.String("err", fmt.Sprintf("%+v", err)))
				debug.PrintStack()
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) disableCacheInDevMode(next http.Handler) http.Handler {
	if app.config.env != "development" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exists := app.session.Exists(r, "clientSessionId")
		if !exists {
			next.ServeHTTP(w, r)
			return
		}

		clientSessionId := app.session.GetString(r, "clientSessionId")
		_, exists = app.hub.GetClient(clientSessionId)
		if !exists {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyIsAuthenticated, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.isAuthenticated(r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}
