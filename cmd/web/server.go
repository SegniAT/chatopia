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

func (app *application) serve(h slog.Handler) error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		ErrorLog:     slog.NewLogLogger(h, slog.LevelError),
	}

	shutdownError := make(chan error)

	go func() {
		// intercept signal, blocking (buffered channel)
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.logger.InfoContext(context.Background(), "shutting down server", slog.String("signal", s.String()))

		// cancel background tasks
		// TODO: this might be bad, srv.Shutdown() already waits for active connections without interrupting
		app.cancel()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		app.logger.InfoContext(context.Background(), "shutting down server", slog.String("signal", s.String()))

		app.wg.Wait()
		shutdownError <- nil
	}()

	app.logger.InfoContext(context.Background(), "starting server", slog.String("addr", srv.Addr), slog.String("env", app.config.env))

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

	app.logger.InfoContext(context.Background(), "stopped server", slog.String("addr", srv.Addr), slog.String("env", app.config.env))

	return nil
}
