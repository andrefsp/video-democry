package chap7

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/andrefsp/video-democry/go/config"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type chap7Handler struct {
	userFactory *userFactory
	roomFactory *roomFactory
}

func (s *chap7Handler) sendMessage(r *room, conn *websocket.Conn, payload interface{}) error {
	if r != nil {
		// Make sure messages to the room are synchronized
		r.messageMutex.Lock()
		defer r.messageMutex.Unlock()
	}

	jData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, jData)
}

func (s *chap7Handler) RegisterHandlers(m *mux.Router, middleware func(h http.HandlerFunc) http.HandlerFunc) {
	m.HandleFunc("/ws", s.RoomWS)
	m.HandleFunc("/rooms", s.OperatorWS)
}

func New(cfg *config.Config) *chap7Handler {
	return &chap7Handler{
		userFactory: newUserFactory(cfg),
		roomFactory: newRoomFactory(cfg),
	}
}
