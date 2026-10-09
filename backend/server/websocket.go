package server

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// wsAuthProtocol is the WebSocket subprotocol browsers use to carry the Clerk
// session token: `new WebSocket(url, ["bearer", token])`. Sending the token
// in Sec-WebSocket-Protocol keeps it out of URLs, proxy logs and history.
const wsAuthProtocol = "bearer"

// wsSessionToken returns the session token for a WebSocket upgrade request,
// read from the Sec-WebSocket-Protocol header or the __session cookie.
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

// newUpgrader returns a WebSocket upgrader that only accepts allowed
// origins and echoes the auth subprotocol (never the token) back.
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
