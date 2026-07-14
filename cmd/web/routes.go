package main

import (
	"io/fs"
	"net/http"

	assets "github.com/SegniAT"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	// TODO: app.rateLimit
	standardMiddleware := alice.New(metricsMiddleware, app.addRequestID, app.session.Enable, app.recoverPanic, app.logRequest, app.disableCacheInDevMode)
	dynamicMiddleware := alice.New(app.authenticate, app.requireAuthentication)
	mux := http.NewServeMux()

	mux.Handle("GET /ping", standardMiddleware.ThenFunc(app.ping))

	mux.Handle("GET /", standardMiddleware.ThenFunc(app.home))
	mux.Handle("GET /about", standardMiddleware.ThenFunc(app.about))

	mux.Handle("POST /chat/{type}", standardMiddleware.ThenFunc(app.chatPost))
	mux.Handle("GET /chat", standardMiddleware.Then(dynamicMiddleware.ThenFunc(app.chat)))

	mux.Handle("GET /live_users", standardMiddleware.ThenFunc(app.liveUsers))

	mux.Handle("GET /chat-ws", standardMiddleware.Then(dynamicMiddleware.ThenFunc(func(w http.ResponseWriter, r *http.Request) {
		app.ServeWs(w, r)
	})))

	assetsFS, err := fs.Sub(assets.PublicContent, "public/assets")
	if err != nil {
		panic(err)
	}

	mux.Handle("GET /assets/", http.StripPrefix("/assets", http.FileServer(http.FS(assetsFS))))
	mux.Handle("GET /favicon.ico", http.RedirectHandler("/assets/favicon.ico", http.StatusMovedPermanently))

	mux.Handle("GET /metrics", metricsHandler())

	return mux
}
