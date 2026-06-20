package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/SegniAT/internal/matchmaking"
	"github.com/SegniAT/internal/redis"
	internalLogger "github.com/SegniAT/logger"
	"github.com/golangcollege/sessions"
)

type contextKey string

const contextKeyIsAuthenticated = contextKey("isAuthenticated")

type application struct {
	config  config
	wg      sync.WaitGroup
	session *sessions.Session
	hub     *matchmaking.Hub
	ctx     context.Context
	cancel  context.CancelFunc
	redis   *redis.Client
}

func main() {
	cfg := loadConfig()
	setupLogger(cfg.env)

	ctx, cancel := context.WithCancel(context.Background())

	redisClient, err := redis.NewClient(redis.Config{
		Addr:     cfg.redisEnv,
		PoolSize: 100,
	})
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	defer redisClient.Close()

	slog.InfoContext(ctx, "connected to redis", "addr", cfg.redisEnv)

	session := sessions.New([]byte(cfg.secret))
	session.Lifetime = 12 * time.Hour
	session.Secure = cfg.env == "production"

	app := &application{
		config:  cfg,
		session: session,
		hub:     matchmaking.NewHub(ctx, redisClient),
		ctx:     ctx,
		cancel:  cancel,
		redis:   redisClient,
	}

	app.hub.Start()
	slog.InfoContext(ctx, "Hub started")

	err = app.serve()

	if err != nil {
		slog.Error(err.Error())
	}
}

func setupLogger(env string) {
	loggerOpts := internalLogger.PrettyHandlerOptions{
		SlogOpts: slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		},
	}

	var logHandler slog.Handler

	switch env {
	case "production":
		logHandler = internalLogger.ContextHandler{
			Handler: slog.NewJSONHandler(os.Stdout, &loggerOpts.SlogOpts),
		}
	default:
		logHandler = internalLogger.ContextHandler{
			Handler: internalLogger.NewPrettyHandler(os.Stdout, loggerOpts),
		}
	}

	slog.SetDefault(slog.New(logHandler))
}
