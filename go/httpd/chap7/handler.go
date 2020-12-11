package chap7

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v3"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/andrefsp/video-democry/go/config"
	"github.com/andrefsp/video-democry/go/httpd/responses"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var messageMutex = sync.Mutex{}

var rooms = sync.Map{}

type chap7Handler struct {
	userFactory *userFactory
	cfg         *config.Config
}

func (s *chap7Handler) sendMessage(conn *websocket.Conn, payload interface{}) error {
	messageMutex.Lock()
	defer messageMutex.Unlock()

	jData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, jData)
}

func (s *chap7Handler) handleICECandidate(r *room, conn *websocket.Conn, messagePayload []byte) error {
	user := r.getUser(conn)

	cm := InICECandidate{}
	if err := json.Unmarshal(messagePayload, &cm); err != nil {
		return err
	}

	if cm.Candidate.Candidate == "" {
		return nil
	}

	if err := user.pc.AddICECandidate(cm.Candidate); err != nil {
		log.Printf("Error adding ICECandidate(%s): (%+v)\n", err.Error(), cm.Candidate)
		return err
	}
	log.Println("Added ICECandidate")

	return nil
}

func (s *chap7Handler) sendAnswer(r *room, conn *websocket.Conn) error {
	user := r.getUser(conn)

	// Answer and respond
	answer, err := user.pc.CreateAnswer(nil)
	if err != nil {
		log.Print("Error: ", err.Error())
		return err
	}

	if err := user.pc.SetLocalDescription(answer); err != nil {
		log.Print("Error: ", err.Error())
		return err
	}

	s.sendMessage(conn, &OutAnswer{
		Uri:    "out/answer",
		ToUser: user, // We are answering to the same user.
		Answer: answer,
	})

	return nil
}

func (s *chap7Handler) handleAnswer(r *room, conn *websocket.Conn, messagePayload []byte) error {
	user := r.getUser(conn)

	om := InAnswer{}
	if err := json.Unmarshal(messagePayload, &om); err != nil {
		return err
	}

	if err := user.pc.SetRemoteDescription(om.Answer); err != nil {
		log.Printf("Error: %s\n", err.Error())
	}

	return nil
}

func (s *chap7Handler) handleOffer(r *room, conn *websocket.Conn, messagePayload []byte) error {
	user := r.getUser(conn)

	om := InOffer{}
	if err := json.Unmarshal(messagePayload, &om); err != nil {
		return err
	}

	log.Printf("ConnectionState from user %s :: %s\n", user.ID, user.pc.ConnectionState())

	if user.pc.ConnectionState() != webrtc.PeerConnectionStateNew {
		// Just reset the Status
		if err := user.pc.SetRemoteDescription(om.Offer); err != nil {
			log.Print("Error: ", err.Error())
			return err
		}
		return s.sendAnswer(r, conn)
	}

	user.pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		log.Printf("Sending ICE candidate to %s\n", user.ID)
		s.sendMessage(conn, &OutICECandidate{
			Uri:       "out/icecandidate",
			ToUser:    user,
			Candidate: c.ToJSON(),
		})
	})

	user.pc.OnTrack(func(t *webrtc.TrackRemote, rec *webrtc.RTPReceiver) {
		log.Printf("Received `%s` `%s` track.\n", t.Kind().String(), t.Codec().MimeType)

		// Handle stream subscriptions
		defer r.handleStreamSubscriptions()

		if t.Kind().String() == "video" {
			//user.video = t
			user.addVideoTrack(t)
			return
		}
		if t.Kind().String() == "audio" {
			//user.audio = t
			user.addAudioTrack(t)
			return
		}
	})

	user.pc.OnNegotiationNeeded(func() {
		log.Printf("Requesting ICE negotiation to %s \n", user.ID)

		s.sendMessage(conn, &OutNegotiationNeeded{
			Uri:    "out/negotiationneeded",
			ToUser: user,
		})
	})

	if err := user.pc.SetRemoteDescription(om.Offer); err != nil {
		log.Print("Error: ", err.Error())
		return err
	}

	s.sendAnswer(r, conn)

	return nil
}

func (s *chap7Handler) handleUserJoin(r *room, conn *websocket.Conn, payload []byte) {
	eventURI := "out/user-join"

	message := InUserJoinMessage{}
	if err := json.Unmarshal(payload, &message); err != nil {
		panic(err)
	}

	user, err := s.userFactory.newUser(message.User)
	if err != nil {
		panic(err)
	}

	r.addUser(conn, user)

	for uconn := range r.users {
		s.sendMessage(uconn, &OutUserEventMessage{
			Uri:   eventURI,
			User:  message.User,
			Users: r.getUserList(),
		})
	}
}

func (s *chap7Handler) handleDisconnection(r *room, conn *websocket.Conn) {
	eventURI := "out/user-left"

	user := r.removeUser(conn)

	for uconn := range r.users {
		s.sendMessage(uconn, &OutUserEventMessage{
			Uri:   eventURI,
			User:  user,
			Users: r.getUserList(),
		})
	}
}

func (s *chap7Handler) handleConnection(roomID string, conn *websocket.Conn) {
	r, loaded := rooms.LoadOrStore(roomID, newRoom())
	room := r.(*room)
	if !loaded {
		room.start()
		log.Println("New room has been created")
	}

	for {
		_, messagePayload, err := conn.ReadMessage()
		if err != nil {
			log.Println("read err:", err)
			s.handleDisconnection(room, conn)
			break
		}

		m := message{}
		if err := json.Unmarshal(messagePayload, &m); err != nil {
			log.Println("read err:", err)
			continue
		}

		switch m.Uri {
		case "in/join":
			s.handleUserJoin(room, conn, messagePayload)
		case "in/icecandidate":
			s.handleICECandidate(room, conn, messagePayload)
		case "in/offer":
			s.handleOffer(room, conn, messagePayload)
		case "in/answer":
			s.handleAnswer(room, conn, messagePayload)
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
		userFactory: &userFactory{
			cfg: cfg,
		},
	}
}
