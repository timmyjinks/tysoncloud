package server

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type TaskRegistry struct {
	mu       sync.Mutex
	registry map[string](chan string)
}

type BuildStatus struct {
	Label  string
	IsDone bool
}

type StatusPage struct {
	Statuses []BuildStatus
}

func NewTaskRegistry() *TaskRegistry {
	return &TaskRegistry{
		registry: make(map[string]chan string),
	}
}

func (app *Application) HandleTaskWS(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	if id == "" {
		http.Error(w, "Id is empty", http.StatusBadRequest)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}
	ws.SetWriteDeadline(time.Now().Add(120 * time.Second))

	defer ws.Close()

	app.TaskRegistry.mu.Lock()
	task := app.TaskRegistry.registry[id]
	app.TaskRegistry.mu.Unlock()

	for message := range task {
		err := ws.WriteJSON(struct {
			Message string `json:"message"`
		}{
			Message: message,
		})
		if err != nil {
			log.Println(err)
		}

		if message == "done" {
			break
		}
	}

	app.TaskRegistry.mu.Lock()
	delete(app.TaskRegistry.registry, id)
	app.TaskRegistry.mu.Unlock()
}
