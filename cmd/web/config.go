package main

import (
	"flag"
	"os"
	"strconv"
)

type config struct {
	port          int
	env           string
	secret        string
	redisAddr     string
	redisPassword string
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func loadConfig() config {
	cfg := config{
		port:          getEnvInt("APP_PORT", 4000),
		env:           getEnv("ENV", "development"),
		secret:        getEnv("SESSION_SECRET", "bP1e8X9y2@c5W3u1Nv7K!r4Lq6QjZ0Fm"),
		redisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		redisPassword: getEnv("REDIS_PASSWORD", "admin"),
	}

	flag.IntVar(&cfg.port, "port", cfg.port, "HTTP network address port")
	flag.StringVar(&cfg.env, "env", cfg.env, "Environment (development|testing|production)")
	flag.StringVar(&cfg.secret, "secret", cfg.secret, "Session secret key")
	flag.StringVar(&cfg.redisAddr, "redis-addr", cfg.redisAddr, "Redis address")
	flag.StringVar(&cfg.redisPassword, "redis-pass", cfg.redisPassword, "Redis password")
	flag.Parse()

	return cfg
}
