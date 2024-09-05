package main

import (
	"github.com/justinas/alice"
	"net/http"
)

func (app *application) routes() http.Handler {
	standardMiddleware := alice.New(app.recoverPanic, app.logRequest, app.disableCacheInDevMode)
	dynamicMiddleware := alice.New(app.session.Enable, app.authenticate, app.requireAuthentication)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", app.home)
	mux.HandleFunc("GET /about", app.about)

	mux.Handle("POST /chat/{type}", app.session.Enable(http.HandlerFunc(app.chatPost)))
	mux.Handle("GET /chat", dynamicMiddleware.ThenFunc(app.chat))

	mux.HandleFunc("GET /live_users", app.liveUsers)

	mux.Handle("GET /chat-ws", dynamicMiddleware.ThenFunc(func(w http.ResponseWriter, r *http.Request) {
		app.ServeWs(w, r)
	}))

	mux.Handle("GET /assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("./public/assets"))))

	return standardMiddleware.Then(mux)
}
