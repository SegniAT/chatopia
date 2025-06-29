package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/SegniAdebaGodsSon/logger"
	"github.com/google/uuid"
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
				slog.Error("panic", "err", err)
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
