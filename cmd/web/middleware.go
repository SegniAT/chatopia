package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/SegniAdebaGodsSon/logger"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

func (app *application) addRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := logger.AppendCtx(r.Context(), slog.String("req_id", uuid.NewString()))
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.logger.InfoContext(r.Context(),
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
				slog.ErrorContext(r.Context(), "panic", slog.Any("err", err))
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
		_, exists = app.hub.Matchmaker.GetClient(clientSessionId)
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
			w.Header().Set("HX-Redirect", "/")
			w.WriteHeader(http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	var (
		mu      sync.Mutex
		clients = make(map[string]*rate.Limiter)
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			w.Header().Set("HX-Trigger-After-Settle", `{"showToast": "Couldn't extract your IP address!"}`)
			w.Header().Set("HX-Redirect", "/")
			w.WriteHeader(http.StatusSeeOther)
			return
		}

		mu.Lock()

		if _, found := clients[ip]; !found {
			clients[ip] = rate.NewLimiter(2, 5)
		}

		if !clients[ip].Allow() {
			mu.Unlock()
			w.Write([]byte("Rate limit exceeded!"))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
