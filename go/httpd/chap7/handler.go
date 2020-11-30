package chap7

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/andrefsp/video-democry/go/config"
	"github.com/andrefsp/video-democry/go/httpd/responses"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// messages
type InfoMessage struct {
	Uri     string `json:"uri"`
	Message string `json:"message"`
}

type message struct {
	Uri string `json:"uri"`
}

type InICECandidate struct {
	FromUser  *user       `json:"from_user"`
	ToUser    *user       `json:"to_user"`
	Candidate interface{} `json:"candidate"`
}

type OutICECandidate struct {
	Uri       string      `json:"uri"`
	FromUser  *user       `json:"from_user"`
	ToUser    *user       `json:"to_user"`
	Candidate interface{} `json:"candidate"`
}

type InOffer struct {
	FromUser *user       `json:"from_user"`
	ToUser   *user       `json:"to_user"`
	Offer    interface{} `json:"offer"`
}

type OutOffer struct {
	Uri      string      `json:"uri"`
	FromUser *user       `json:"from_user"`
	ToUser   *user       `json:"to_user"`
	Offer    interface{} `json:"offer"`
}

type InAnswer struct {
	FromUser *user       `json:"from_user"`
	ToUser   *user       `json:"to_user"`
	Answer   interface{} `json:"answer"`
}

type OutAnswer struct {
	Uri      string      `json:"uri"`
	FromUser *user       `json:"from_user"`
	ToUser   *user       `json:"to_user"`
	Answer   interface{} `json:"answer"`
}

type InUserJoinMessage struct {
	User *user `json:"user"`
}

type OutRoomEventMessage struct {
	Uri   string  `json:"uri"`
	User  *user   `json:"user"`
	Users []*user `json:"room_users"`
}

// models
type user struct {
	Username string `json:"username"`
	// StreamID string `json:"stream_id"`
}

var rooms = map[string]*room{}

type chap7Handler struct {
	cfg *config.Config
}

func (s *chap7Handler) sendMessage(conn *websocket.Conn, payload interface{}) error {
	jData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, jData)
}

func (s *chap7Handler) handleDisconnection(r *room, conn *websocket.Conn) {
}

func (s *chap7Handler) handleConnection(roomID string, conn *websocket.Conn) {
	for {
		_, messagePayload, err := conn.ReadMessage()
		if err != nil {
			log.Println("read err:", err)
			break
		}

		m := message{}
		if err := json.Unmarshal(messagePayload, &m); err != nil {
			log.Println("read err:", err)
			continue
		}

		switch m.Uri {
		case "in/join":
		case "in/icecandidate":
		case "in/offer":
		case "in/answer":
		case "in/pong":
		default:
			s.sendMessage(conn, &InfoMessage{
				Uri: "out/error", Message: "Message uri not recognized",
			})

			log.Println("No handler for message type: ", m.Uri)
		}
	}
}

func (s *chap7Handler) Handler(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		responses.Send(w, http.StatusBadRequest, responses.NewError("room not present on request"))
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		responses.Send(w, http.StatusBadRequest, responses.NewError(err.Error()))
		return
	}

	go s.handleConnection(roomID, c)
}

func (s *chap7Handler) RegisterHandlers(m *mux.Router, middleware func(h http.HandlerFunc) http.HandlerFunc) {
	m.HandleFunc("/ws", s.Handler)
}

func New(cfg *config.Config) *chap7Handler {
	return &chap7Handler{
		cfg: cfg,
	}
}
