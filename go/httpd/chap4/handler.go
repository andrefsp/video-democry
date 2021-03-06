package chap4

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
	User      *user       `json:"user"`
	Candidate interface{} `json:"candidate"`
}

type InOffer struct {
	User  *user       `json:"user"`
	Offer interface{} `json:"offer"`
}

type OutOffer struct {
	Uri   string      `json:"uri"`
	User  *user       `json:"user"`
	Offer interface{} `json:"offer"`
}

type InAnswer struct {
	User     *user       `json:"user"`
	DestUser *user       `json:"dest_user"`
	Answer   interface{} `json:"answer"`
}

type OutAnswer struct {
	Uri    string      `json:"uri"`
	User   *user       `json:"user"`
	Answer interface{} `json:"answer"`
}

type InUserJoinMessage struct {
	User *user `json:"user"`
}

type OutRoomEventMessage struct {
	Uri   string  `json:"uri"`
	User  *user   `json:"user"`
	Users []*user `json:"room_users"`
}

type OutICECandidate struct {
	Uri       string      `json:"uri"`
	User      *user       `json:"user"`
	Candidate interface{} `json:"candidate"`
}

// models
type user struct {
	Username string `json:"username"`
	StreamID string `json:"stream_id"`
}

type room struct {
	users map[*websocket.Conn]*user
}

func (r *room) getUserList() []*user {
	users := []*user{}
	for p := range r.users {
		users = append(users, r.users[p])
	}
	return users
}

var rooms = map[string]*room{}

type chap4Handler struct {
	cfg *config.Config
}

func (s *chap4Handler) sendMessage(conn *websocket.Conn, payload interface{}) error {
	jData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, jData)
}

func (s *chap4Handler) handleUserJoined(r *room, conn *websocket.Conn, messagePayload []byte) error {
	uj := InUserJoinMessage{}
	if err := json.Unmarshal(messagePayload, &uj); err != nil {
		return err
	}
	if _, ok := r.users[conn]; ok {
		s.sendMessage(conn, &InfoMessage{Uri: "out/info", Message: "User already joined"})
		return nil
	}
	r.users[conn] = uj.User
	s.pushRoomStatus(r, uj.User, "out/user-join")
	return nil
}

func (s *chap4Handler) handleICECandidate(r *room, conn *websocket.Conn, messagePayload []byte) error {
	cm := InICECandidate{}
	if err := json.Unmarshal(messagePayload, &cm); err != nil {
		return err
	}

	for conn, user := range r.users {
		if user.Username == cm.User.Username {
			// ICE candidates are not pushed to the same connection
			continue
		}
		err := s.sendMessage(conn, &OutICECandidate{
			Uri:       "out/icecandidate",
			User:      cm.User,
			Candidate: cm.Candidate,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *chap4Handler) handleOffer(r *room, conn *websocket.Conn, messagePayload []byte) error {
	om := InOffer{}
	if err := json.Unmarshal(messagePayload, &om); err != nil {
		return err
	}

	for conn, user := range r.users {
		if user.Username == om.User.Username {
			// ICE candidates are not pushed to the same connection
			continue
		}
		err := s.sendMessage(conn, &OutOffer{
			Uri:   "out/offer",
			User:  om.User,
			Offer: om.Offer,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *chap4Handler) handleAnswer(r *room, conn *websocket.Conn, messagePayload []byte) error {
	am := InAnswer{}
	if err := json.Unmarshal(messagePayload, &am); err != nil {
		return err
	}

	// Answer is only sent to one unique user
	for conn, user := range r.users {
		if user.Username != am.DestUser.Username {
			// ICE candidates are not pushed to the same connection
			continue
		}
		err := s.sendMessage(conn, &OutAnswer{
			Uri:    "out/answer",
			User:   am.User,
			Answer: am.Answer,
		})
		if err != nil {
			return err
		}
		break
	}

	return nil
}

func (s *chap4Handler) pushRoomStatus(r *room, u *user, uri string) error {
	// send current users on the call

	payload := &OutRoomEventMessage{
		Uri:   uri,
		User:  u,
		Users: r.getUserList(),
	}

	for conn := range r.users {
		if err := s.sendMessage(conn, payload); err != nil {
			log.Println("write err:", err)
			return err
		}
	}
	return nil
}

func (s *chap4Handler) handleDisconnection(r *room, conn *websocket.Conn) {
	defer conn.Close()
	log.Printf("Connection went away %+v \n", conn)

	u, ok := r.users[conn]
	if !ok {
		return
	}
	delete(r.users, conn)
	s.pushRoomStatus(r, u, "out/user-left")
}

func (s *chap4Handler) handleConnection(roomID string, conn *websocket.Conn) {
	var r *room
	r, ok := rooms[roomID]
	if !ok {
		log.Printf("Created room %s \n", roomID)
		r = &room{
			users: map[*websocket.Conn]*user{},
		}
		rooms[roomID] = r
	}

	for {
		_, messagePayload, err := conn.ReadMessage()
		if err != nil {
			log.Println("read err:", err)
			s.handleDisconnection(r, conn)
			break
		}

		m := message{}
		if err := json.Unmarshal(messagePayload, &m); err != nil {
			log.Println("read err:", err)
			continue
		}

		switch m.Uri {
		case "in/join":
			s.handleUserJoined(r, conn, messagePayload)
		case "in/icecandidate":
			s.handleICECandidate(r, conn, messagePayload)
		case "in/offer":
			s.handleOffer(r, conn, messagePayload)
		case "in/answer":
			s.handleAnswer(r, conn, messagePayload)
		default:
			s.sendMessage(conn, &InfoMessage{
				Uri: "out/error", Message: "Message uri not recognized",
			})

			log.Println("No handler for message type: ", m.Uri)
		}

	}
}

func (s *chap4Handler) Handler(w http.ResponseWriter, r *http.Request) {
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

func (s *chap4Handler) RegisterHandlers(m *mux.Router, middleware func(h http.HandlerFunc) http.HandlerFunc) {
	m.HandleFunc("/endpoint", s.Handler)
}

func New(cfg *config.Config) *chap4Handler {
	return &chap4Handler{
		cfg: cfg,
	}
}
