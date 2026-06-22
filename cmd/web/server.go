package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  2 * time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}

	shutdownError := make(chan error)

	go func() {
		// intercept signal, blocking (buffered channel)
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		slog.InfoContext(context.Background(), "shutting down server", slog.String("signal", s.String()))

		app.hub.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		slog.InfoContext(context.Background(), "shutting down server", slog.String("signal", s.String()))

		shutdownError <- nil
	}()

	slog.InfoContext(context.Background(), "starting server", slog.String("addr", srv.Addr), slog.String("env", app.config.env))

	//err := srv.ListenAndServeTLS("/home/sgngodsson/Desktop/Chatopia/tls/cert.pem", "/home/sgngodsson/Desktop/Chatopia/tls/key.pem")
	err := srv.ListenAndServe()

	// ErrServerClosed is returned immidiately after calling Shutdown() ...
	// an indication of graceful shutdown starting, catch other errors not this 'error'
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	slog.InfoContext(context.Background(), "stopped server", slog.String("addr", srv.Addr), slog.String("env", app.config.env))

	return nil
}
