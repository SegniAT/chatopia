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

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	component := templates.Home("Home")
	component.Render(r.Context(), w)
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	component := templates.About("About Chatopia")
	component.Render(r.Context(), w)
}

func (app *application) chatPost(w http.ResponseWriter, r *http.Request) {
	interests, interestsErr := app.validateInterests(w, r)
	r.ParseForm()

	chatType := r.PathValue("type")
	if chatType != "video" && chatType != "text" {
		component := templates.InterestInput([]string{}, fmt.Errorf("invalid chat type"))
		component.Render(r.Context(), w)
		return
	}

	if interestsErr != nil {
		component := templates.InterestInput(interests, interestsErr)
		component.Render(r.Context(), w)
		return
	}

	clientSessionId := uuid.NewString()

	client := &internalWS.Client{
		SessionID: clientSessionId,
		ChatType:  chatType,
		Interests: interests,
	}

	app.hub.OnlineClients.StoreClient(client.SessionID, client)

	slog.Info("client registered", "sessionID:", client.SessionID, "interests:", client.Interests)

	app.session.Put(r, "clientSessionId", clientSessionId)

	w.Header().Set("HX-Redirect", "/chat")
}

func (app *application) chat(w http.ResponseWriter, r *http.Request) {
	sessionId := app.session.GetString(r, "clientSessionId")

	client, bool := app.hub.OnlineClients.GetClient(sessionId)
	if !bool {
		w.Header().Set("HX-Redirect", "/")
		return
	}

	chatType := client.ChatType
	component := templates.TextChat(fmt.Sprintf("Chat | %s", strings.ToUpper(chatType)))
	component.Render(r.Context(), w)
}

func (app *application) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("(serveWs)", "error", err.Error())
	}

	sessionId := app.session.GetString(r, "clientSessionId")

	client, bool := app.hub.OnlineClients.GetClient(sessionId)
	if !bool {
		w.Header().Set("HX-Redirect", "/")
		return
	}

	client.Conn = conn
	client.Hub = app.hub
	client.Send = make(chan []byte, 256)

	app.hub.Register <- client

	go client.WritePump()
	go client.ReadPump()

}
