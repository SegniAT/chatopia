package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/SegniAT/internal/matchmaking"
	"github.com/SegniAT/internal/metrics"
	"github.com/SegniAT/ui/templates"
	"github.com/google/uuid"
	gorillaWS "github.com/gorilla/websocket"
)

var upgrader = gorillaWS.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (app *application) ping(w http.ResponseWriter, _ *http.Request) {
	_, err := w.Write([]byte("OK"))
	if err != nil {
		return
	}
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	metrics.VisitsTotal.WithLabelValues("home").Inc()

	component := templates.Home("Home", false, []string{}, app.hub.ClientCount())
	component.Render(r.Context(), w)
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	metrics.VisitsTotal.WithLabelValues("about").Inc()

	component := templates.About("About")
	component.Render(r.Context(), w)
}

func (app *application) liveUsers(w http.ResponseWriter, r *http.Request) {
	component := templates.LiveUsers(app.hub.ClientCount())
	component.Render(r.Context(), w)
}

func (app *application) chatPost(w http.ResponseWriter, r *http.Request) {
	chatType := r.PathValue("type")
	if !(chatType == "video" || chatType == "text") {
		component := templates.InterestInput([]string{}, fmt.Errorf("invalid chat type"))
		component.Render(r.Context(), w)
		return
	}

	isStrict, interests, interestsErr := app.validateInterests(w, r)
	r.ParseForm()

	if interestsErr != nil {
		component := templates.InterestInput(interests, interestsErr)
		component.Render(r.Context(), w)
		return
	}

	client := matchmaking.NewClient(uuid.NewString(), chatType, isStrict, interests, app.hub)

	err := app.hub.RegisterClient(client)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to register client with Hub", slog.String("error", err.Error()))
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	slog.InfoContext(r.Context(),
		"client registered to HUB",
		slog.Any("client", client),
	)

	app.session.Put(r, "clientSessionId", client.SessionID)

	w.Header().Set("HX-Redirect", "/chat")
	w.WriteHeader(http.StatusSeeOther)
}

func (app *application) chat(w http.ResponseWriter, r *http.Request) {
	metrics.VisitsTotal.WithLabelValues("chat").Inc()

	sessionId := app.session.GetString(r, "clientSessionId")

	client, ok := app.hub.GetClient(sessionId)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	chatType := client.ChatType
	component := templates.Chat(fmt.Sprintf("Chat | %s", strings.ToUpper(chatType)), chatType == "video", client.Interests, app.hub.ClientCount(), app.config.turnUser, app.config.turnCredential)
	component.Render(r.Context(), w)
}

func (app *application) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "serveWs error", slog.String("error", err.Error()))
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	sessionId := app.session.GetString(r, "clientSessionId")

	client, ok := app.hub.GetClient(sessionId)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	client.Conn = conn
	client.RemoteIP = r.RemoteAddr
	client.ConnStartedAt = time.Now()

	app.hub.StartMatchmaking(client)
	app.hub.ClientConnected(client)
}
