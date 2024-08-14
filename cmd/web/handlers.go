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
	interests := []string{}

	exists := app.session.Exists(r, "clientSessionId")
	if exists {
		sessionId := app.session.GetString(r, "clientSessionId")
		client, exists := app.hub.OnlineClients.GetClient(sessionId)
		if exists {
			interests = client.Interests
		}
	}

	component := templates.Home("Home", interests)
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
	w.WriteHeader(http.StatusSeeOther)
}

func (app *application) chat(w http.ResponseWriter, r *http.Request) {
	sessionId := app.session.GetString(r, "clientSessionId")
	fmt.Println("chat handler")

	client, bool := app.hub.OnlineClients.GetClient(sessionId)
	if !bool {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	chatType := client.ChatType
	fmt.Println("chatType: ", chatType)
	component := templates.Chat(fmt.Sprintf("Chat | %s", strings.ToUpper(chatType)), chatType == "video")
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
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	client.Conn = conn
	client.Hub = app.hub
	client.Send = make(chan []byte, 256)

	app.hub.Register <- client

	go client.WritePump()
	go client.ReadPump()

}
