package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	internalWS "github.com/SegniAdebaGodsSon/internal/websocket"
	internalLogger "github.com/SegniAdebaGodsSon/logger"
	"github.com/golangcollege/sessions"
)

func newTestApplication(_ *testing.T) *application {
	var cfg config

	cfg.env = "testing"
	cfg.secret = "bP1e8X9y2@c5W3u1Nv7K!r4Lq6QjZ0Fm"

	session := sessions.New([]byte(cfg.secret))
	session.Lifetime = 12 * time.Hour
	session.Secure = cfg.env == "production"

	ctx, cancel := context.WithCancel(context.Background())

	loggerOpts := internalLogger.PrettyHandlerOptions{
		SlogOpts: slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		},
	}

	logHandler := internalLogger.ContextHandler{
		Handler: internalLogger.NewPrettyHandler(os.Stdout, loggerOpts),
	}
	logger := slog.New(logHandler)
	return &application{
		config:  cfg,
		session: session,
		hub:     internalWS.NewHub(ctx, logger),
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger,
	}
}

type testServer struct {
	*httptest.Server
}

func newTestServer(t *testing.T, h http.Handler) *testServer {
	ts := httptest.NewServer(h)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}

	ts.Client().Jar = jar

	ts.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &testServer{ts}
}

func (ts *testServer) get(t *testing.T, urlPath string) (int, http.Header, []byte) {
	rs, err := ts.Client().Get(ts.URL + urlPath)
	if err != nil {
		t.Fatal(err)
	}

	defer rs.Body.Close()
	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	return rs.StatusCode, rs.Header, body
}

func (ts *testServer) postForm(t *testing.T, urlPath string, form url.Values) (int, http.Header, []byte) {
	rs, err := ts.Client().PostForm(ts.URL+urlPath, form)
	if err != nil {
		t.Fatal(err)
	}

	defer rs.Body.Close()
	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	return rs.StatusCode, rs.Header, body
}
