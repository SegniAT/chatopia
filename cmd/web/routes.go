package main

import (
	"github.com/justinas/alice"
	"net/http"
)

func (app *application) routes() http.Handler {
	standardMiddleware := alice.New(app.addRequestID, app.session.Enable, app.recoverPanic, app.logRequest, app.disableCacheInDevMode)
	dynamicMiddleware := alice.New(app.authenticate, app.requireAuthentication)
	mux := http.NewServeMux()

	mux.Handle("GET /", standardMiddleware.ThenFunc(app.home))
	mux.Handle("GET /about", standardMiddleware.ThenFunc(app.about))

	mux.Handle("POST /chat/{type}", standardMiddleware.ThenFunc(app.chatPost))
	mux.Handle("GET /chat", standardMiddleware.Then(dynamicMiddleware.ThenFunc(app.chat)))

	mux.Handle("GET /live_users", standardMiddleware.ThenFunc(app.liveUsers))

	mux.Handle("GET /chat-ws", standardMiddleware.Then(dynamicMiddleware.ThenFunc(func(w http.ResponseWriter, r *http.Request) {
		app.ServeWs(w, r)
	})))

	mux.Handle("GET /assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("./public/assets"))))

	mux.Handle("GET /ping", standardMiddleware.ThenFunc(app.ping))

	return mux
}
