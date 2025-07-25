package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"sync"
	"time"

	internalWS "github.com/SegniAdebaGodsSon/internal/websocket"
	internalLogger "github.com/SegniAdebaGodsSon/logger"
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
	ctx     context.Context
	cancel  context.CancelFunc
	logger  *slog.Logger
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "HTTP network address port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|testing|production)")
	flag.StringVar(&cfg.secret, "secret", "bP1e8X9y2@c5W3u1Nv7K!r4Lq6QjZ0Fm", "Secret key")

	flag.Parse()

	logger, logHandler, err := setupLogger(&cfg)
	if err != nil {
		log.Fatalf("error setting up logging %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	session := sessions.New([]byte(cfg.secret))
	session.Lifetime = 12 * time.Hour
	session.Secure = cfg.env == "production"

	app := &application{
		config:  cfg,
		session: session,
		hub:     internalWS.NewHub(ctx, logger),
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger,
	}

	app.logger.InfoContext(ctx, "Starting Hub")
	go app.hub.Run()

	err = app.serve(logHandler)

	if err != nil {
		slog.Error(err.Error())
	}
}

func setupLogger(cfg *config) (*slog.Logger, slog.Handler, error) {
	loggerOpts := internalLogger.PrettyHandlerOptions{
		SlogOpts: slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		},
	}

	var logHandler slog.Handler

	switch cfg.env {
	case "production":
		file, err := os.OpenFile("./app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file %v", err)
		}

		logHandler = internalLogger.ContextHandler{
			Handler: slog.NewJSONHandler(file, &loggerOpts.SlogOpts),
		}
	default:
		logHandler = internalLogger.ContextHandler{
			Handler: internalLogger.NewPrettyHandler(os.Stdout, loggerOpts),
		}
	}
	return slog.New(logHandler), logHandler, nil
}
