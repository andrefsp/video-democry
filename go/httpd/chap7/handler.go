package chap7

/*
import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/andrefsp/video-democry/go/config"
	"github.com/andrefsp/video-democry/go/httpd/responses"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// models
type user struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	StreamID string `json:"streamID"`

	pc *webrtc.PeerConnection
}

func (u *user) setPeerConnection() {}

var rooms = sync.Map{}

type chap7Handler struct {
	cfg *config.Config
}

func (s *chap7Handler) newPeerConnection(offer webrtc.SessionDescription) (*webrtc.PeerConnection, error) {
	mediaEngine := webrtc.MediaEngine{}
	if err := mediaEngine.PopulateFromSDP(offer); err != nil {
		return nil, err
	}

	return webrtc.
		NewAPI(webrtc.WithMediaEngine(mediaEngine)).
		NewPeerConnection(webrtc.Configuration{
			ICETransportPolicy: webrtc.ICETransportPolicyRelay,
			ICEServers: []webrtc.ICEServer{
				//{
				//	URLs: []string{"stun:stun.l.google.com:19302"},
				//},
				{
					URLs:       []string{s.cfg.TurnServerAddr},
					Credential: "thiskey",
					Username:   "thisuser",
				},
			},
		})
}

func (s *chap7Handler) sendMessage(conn *websocket.Conn, payload interface{}) error {
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

func (s *chap7Handler) handleOffer(r *room, conn *websocket.Conn, messagePayload []byte) error {
	user := r.getUser(conn)

	om := InOffer{}
	if err := json.Unmarshal(messagePayload, &om); err != nil {
		return err
	}

	pc, err := s.newPeerConnection(om.Offer)
	if err != nil {
		log.Print("Error creating Peer connection: ", err.Error())
		return err
	}

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		log.Println("Sending ICE candidate.")
		s.sendMessage(conn, &OutICECandidate{
			Uri:       "out/icecandidate",
			ToUser:    user,
			Candidate: c.ToJSON(),
		})
	})

	pc.OnTrack(func(t *webrtc.Track, r *webrtc.RTPReceiver) {
		log.Printf("We got a track:: %+v", t)
		log.Printf("We got a receiver:: %+v", r)

		if _, err := pc.AddTrack(t); err != nil {
			log.Println("Error: ", err.Error())
		}
	})

	if err := pc.SetRemoteDescription(om.Offer); err != nil {
		log.Print("Error: ", err.Error())
		return err
	}

	// Answer and respond
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Print("Error: ", err.Error())
		return err
	}

	if err := pc.SetLocalDescription(answer); err != nil {
		log.Print("Error: ", err.Error())
		return err
	}

	user.pc = pc

	s.sendMessage(conn, &OutAnswer{
		Uri:    "out/answer",
		ToUser: user, // We are answering to the same user.
		Answer: answer,
	})

	return nil
}

func (s *chap7Handler) handleUserJoin(r *room, conn *websocket.Conn, payload []byte) {
	eventURI := "out/user-join"

	message := InUserJoinMessage{}
	if err := json.Unmarshal(payload, &message); err != nil {
		panic(err)
	}

	r.addUser(conn, message.User)

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

	user := r.getUser(conn)

	r.removeUser(conn, user)

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
*/
