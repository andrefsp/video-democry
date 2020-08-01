package chap5

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/andrefsp/video-democry/go/httpd/responses"
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
	StreamID string `json:"stream_id"`
}

type room struct {
	users map[*websocket.Conn]*user

	ticker <-chan time.Time
	stop   <-chan struct{}
}

func (r *room) start() {
	for {
		select {
		case <-r.ticker:
			for conn := range r.users {
				conn.WriteMessage(websocket.TextMessage, []byte(`{"uri":"out/ping"}`))
			}
		case <-r.stop:
			break
		}
	}
}

func (r *room) getUserList() []*user {
	users := []*user{}
	for p := range r.users {
		users = append(users, r.users[p])
	}
	return users
}

func newRoom() *room {
	r := &room{
		users:  map[*websocket.Conn]*user{},
		ticker: time.NewTicker(15 * time.Second).C,
		stop:   make(<-chan struct{}, 1),
	}
	go r.start()
	return r
}

var rooms = map[string]*room{}

type chap5Handler struct{}

func (s *chap5Handler) sendMessage(conn *websocket.Conn, payload interface{}) error {
	jData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, jData)
}

func (s *chap5Handler) handleUserJoined(r *room, conn *websocket.Conn, messagePayload []byte) error {
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

func (s *chap5Handler) handleICECandidate(r *room, conn *websocket.Conn, messagePayload []byte) error {
	cm := InICECandidate{}
	if err := json.Unmarshal(messagePayload, &cm); err != nil {
		return err
	}
	for conn, user := range r.users {
		if user.Username != cm.ToUser.Username {
			// ICE candidates are not pushed to the same connection
			continue
		}
		err := s.sendMessage(conn, &OutICECandidate{
			Uri:       "out/icecandidate",
			ToUser:    cm.ToUser,
			FromUser:  cm.FromUser,
			Candidate: cm.Candidate,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *chap5Handler) handleOffer(r *room, conn *websocket.Conn, messagePayload []byte) error {
	om := InOffer{}
	if err := json.Unmarshal(messagePayload, &om); err != nil {
		return err
	}

	for conn, user := range r.users {
		if user.Username != om.ToUser.Username {
			// ICE candidates are not pushed to the same connection
			continue
		}
		err := s.sendMessage(conn, &OutOffer{
			Uri:      "out/offer",
			ToUser:   om.ToUser,
			FromUser: om.FromUser,
			Offer:    om.Offer,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *chap5Handler) handleAnswer(r *room, conn *websocket.Conn, messagePayload []byte) error {
	am := InAnswer{}
	if err := json.Unmarshal(messagePayload, &am); err != nil {
		return err
	}

	// Answer is only sent to one unique user
	for conn, user := range r.users {
		if user.Username != am.ToUser.Username {
			// ICE candidates are not pushed to the same connection
			continue
		}
		err := s.sendMessage(conn, &OutAnswer{
			Uri:      "out/answer",
			ToUser:   am.ToUser,
			FromUser: am.FromUser,
			Answer:   am.Answer,
		})
		if err != nil {
			return err
		}
		break
	}

	return nil
}

func (s *chap5Handler) pushRoomStatus(r *room, u *user, uri string) error {
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

func (s *chap5Handler) handleDisconnection(r *room, conn *websocket.Conn) {
	defer conn.Close()
	log.Print("Connection went away...")

	u, ok := r.users[conn]
	if !ok {
		return
	}
	delete(r.users, conn)
	s.pushRoomStatus(r, u, "out/user-left")
}

func (s *chap5Handler) handleConnection(roomID string, conn *websocket.Conn) {
	var r *room
	r, ok := rooms[roomID]
	if !ok {
		log.Printf("Created room %s \n", roomID)
		r = newRoom()

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
		case "in/pong":
		default:
			s.sendMessage(conn, &InfoMessage{
				Uri: "out/error", Message: "Message uri not recognized",
			})

			log.Println("No handler for message type: ", m.Uri)
		}

	}
}

func (s *chap5Handler) Handler(w http.ResponseWriter, r *http.Request) {
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

func New() *chap5Handler {
	return &chap5Handler{}
}
