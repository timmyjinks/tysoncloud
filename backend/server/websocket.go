package server

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

const wsAuthProtocol = "bearer"

func wsSessionToken(r *http.Request) string {
	protocols := websocket.Subprotocols(r)
	for i := 0; i+1 < len(protocols); i++ {
		if protocols[i] == wsAuthProtocol {
			return strings.TrimSpace(protocols[i+1])
		}
	}
	if cookie, err := r.Cookie("__session"); err == nil {
		return cookie.Value
	}
	return ""
}

func (app *Application) newUpgrader() websocket.Upgrader {
	allowedOrigins := parseAllowedOrigins(app.Config.Server.AllowedOrigins)
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		Subprotocols:    []string{wsAuthProtocol},
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			return allowedOrigins[origin]
		},
	}
}
