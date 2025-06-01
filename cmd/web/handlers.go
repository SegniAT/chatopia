package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	internalWS "github.com/SegniAdebaGodsSon/internal/websocket"
	"github.com/SegniAdebaGodsSon/ui/templates"
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
	var isStrict bool
	var interests []string
	sessionId := app.session.GetString(r, "clientSessionId")
	client, ok := app.hub.OnlineClients.GetClient(sessionId)
	if ok {
		isStrict = client.IsStrict
		interests = client.Interests
	}
	component := templates.Home("Home", isStrict, interests)
	component.Render(r.Context(), w)
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	component := templates.About("About")
	component.Render(r.Context(), w)
}

func (app *application) liveUsers(w http.ResponseWriter, r *http.Request) {
	component := templates.LiveUsers(app.hub.OnlineClients.Size())
	component.Render(r.Context(), w)
}

func (app *application) chatPost(w http.ResponseWriter, r *http.Request) {
	chatType := r.PathValue("type")
	if !(chatType == "video" || chatType == "text") {
		w.WriteHeader(http.StatusBadRequest)
		component := templates.InterestInput([]string{}, fmt.Errorf("invalid chat type"))
		component.Render(r.Context(), w)
		return
	}

	isStrict, interests, interestsErr := app.validateInterests(w, r)
	r.ParseForm()

	if interestsErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		component := templates.InterestInput(interests, interestsErr)
		component.Render(r.Context(), w)
		return
	}

	clientSessionId := uuid.NewString()

	client := internalWS.NewClient(clientSessionId, chatType, isStrict, interests, app.hub)

	app.hub.OnlineClients.StoreClient(client.SessionID, client)

	app.logger.InfoContext(r.Context(),
		"client registered to store",
		slog.Any("client", client),
	)

	app.session.Put(r, "clientSessionId", clientSessionId)

	w.Header().Set("HX-Redirect", "/chat")
	w.WriteHeader(http.StatusSeeOther)
}

func (app *application) chat(w http.ResponseWriter, r *http.Request) {
	sessionId := app.session.GetString(r, "clientSessionId")

	client, ok := app.hub.OnlineClients.GetClient(sessionId)
	if !ok {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	chatType := client.ChatType
	component := templates.Chat(fmt.Sprintf("Chat | %s", strings.ToUpper(chatType)), chatType == "video", client.Interests)
	component.Render(r.Context(), w)
}

func (app *application) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.logger.ErrorContext(r.Context(), "serveWs error", slog.String("error", err.Error()))
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusSeeOther)
	}

	sessionId := app.session.GetString(r, "clientSessionId")

	client, ok := app.hub.OnlineClients.GetClient(sessionId)
	if !ok {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	client.Conn = conn

	err = app.hub.RegisterClient(client)
	if err != nil {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusInternalServerError)
	}

	app.logger.InfoContext(r.Context(),
		"client registered to HUB",
		slog.Any("client", client),
	)

	go client.WritePump()
	go client.ReadPump()
}
