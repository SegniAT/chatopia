package main

import (
	"flag"
	"log/slog"
	"sync"
	"time"

	internalWS "github.com/SegniAdebaGodsSon/internal/websocket"
	"github.com/golangcollege/sessions"
)

type config struct {
	port   int
	env    string
	secret string
}

type contextKey string

const contextKeyIsAuthenticated = contextKey("isAuthenticated")

type application struct {
	config  config
	wg      sync.WaitGroup
	session *sessions.Session
	hub     *internalWS.Hub
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "HTTP network address port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.secret, "secret", "bP1e8X9y2@c5W3u1Nv7K!r4Lq6QjZ0Fm", "Secret key")

	flag.Parse()

	session := sessions.New([]byte(cfg.secret))
	session.Lifetime = 12 * time.Hour
	session.Secure = cfg.env == "production"

	app := &application{
		config:  cfg,
		session: session,
		hub: &internalWS.Hub{
			OnlineClients: internalWS.NewOnlineClientsStore(),
			Recieve:       make(chan *internalWS.Message),
			Register:      make(chan *internalWS.Client),
			Unregister:    make(chan *internalWS.Client),
		},
	}

	app.background(func() {
		slog.Info("Starting Hub")
		app.hub.Run()
	})

	err := app.serve()

	if err != nil {
		slog.Error(err.Error())
	}
}
