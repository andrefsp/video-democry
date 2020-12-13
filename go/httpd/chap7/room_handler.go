package chap7

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/pion/webrtc/v3"

	"github.com/gorilla/websocket"

	"github.com/andrefsp/video-democry/go/httpd/responses"
)

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
	//log.Printf("Added ICECandidate from user `%s`", user.ID)

	return nil
}

func (s *chap7Handler) sendAnswer(r *room, conn *websocket.Conn, offer webrtc.SessionDescription) error {
	user := r.getUser(conn)

	if err := user.pc.SetRemoteDescription(offer); err != nil {
		log.Print("Error: ", err.Error())
		return err
	}

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

	s.sendMessage(r, conn, &OutAnswer{
		Uri:    "out/answer",
		ToUser: user, // We are answering to the same user.
		Answer: answer,
	})

	log.Printf("Answer sent to user %s", user.ID)

	return nil
}

func (s *chap7Handler) handleAnswer(r *room, conn *websocket.Conn, messagePayload []byte) error {
	user := r.getUser(conn)

	om := InAnswer{}
	if err := json.Unmarshal(messagePayload, &om); err != nil {
		log.Printf("Error: %s\n", err.Error())
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

	log.Printf("Offer from user `%s` ConnectionState: `%s`", user.ID, user.pc.ConnectionState())

	if user.pc.ConnectionState() != webrtc.PeerConnectionStateNew {
		// Just reset the Status
		return s.sendAnswer(r, conn, om.Offer)
	}

	user.pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		//log.Printf("Sending ICE candidate to `%s`", user.ID)
		s.sendMessage(r, conn, &OutICECandidate{
			Uri:       "out/icecandidate",
			ToUser:    user,
			Candidate: c.ToJSON(),
		})
	})

	user.pc.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		switch s {
		case webrtc.ICEConnectionStateFailed:
			fallthrough
		case webrtc.ICEConnectionStateConnected:
			fallthrough
		case webrtc.ICEConnectionStateCompleted:
			fallthrough
		case webrtc.ICEConnectionStateClosed:
			fallthrough
		case webrtc.ICEConnectionStateDisconnected:
			log.Printf("ICE state `%s` with user `%s`", s.String(), user.ID)
		}
	})

	user.pc.OnTrack(func(t *webrtc.TrackRemote, rec *webrtc.RTPReceiver) {
		log.Printf("Received track: `%s` mimetype: `%s`.\n", t.Kind().String(), t.Codec().MimeType)

		// Handle stream subscriptions
		defer r.handleStreamSubscriptions()

		if t.Kind().String() == "video" {
			user.addVideoTrack(t)
			return
		}
		if t.Kind().String() == "audio" {
			user.addAudioTrack(t)
			return
		}
	})

	user.pc.OnNegotiationNeeded(func() {
		offer, err := user.pc.CreateOffer(nil)
		if err != nil {
			return
		}

		if err := user.pc.SetLocalDescription(offer); err != nil {
			return
		}

		s.sendMessage(r, conn, &OutOffer{
			Uri:    "out/offer",
			ToUser: user,
			Offer:  offer,
		})
		log.Printf("Requested ICE negotiation to %s \n", user.ID)
	})

	return s.sendAnswer(r, conn, om.Offer)
}

func (s *chap7Handler) handleUserJoin(r *room, conn *websocket.Conn, payload []byte) error {
	eventURI := "out/user-join"

	message := InUserJoinMessage{}
	if err := json.Unmarshal(payload, &message); err != nil {
		panic(err)
	}

	user, err := s.userFactory.newUser(message.User)
	if err != nil {
		panic(err)
	}

	if _, err = r.addUser(conn, user); err != nil {
		return s.sendMessage(r, conn, &InfoMessage{
			Uri:     "out/error",
			Message: err.Error(),
		})
	}

	for _, uconn := range r.getUserConnections() {
		s.sendMessage(r, uconn, &OutUserEventMessage{
			Uri:   eventURI,
			User:  message.User,
			Users: r.getUserList(),
		})
	}
	return nil
}

func (s *chap7Handler) handleDisconnection(r *room, conn *websocket.Conn) {
	eventURI := "out/user-left"

	user := r.removeUser(conn)

	for _, uconn := range r.getUserConnections() {
		s.sendMessage(r, uconn, &OutUserEventMessage{
			Uri:   eventURI,
			User:  user,
			Users: r.getUserList(),
		})
	}

	if s.roomFactory.deleteIfEmpty(r) {
		log.Printf("Room `%s` has been deleted.", r.ID)
	}
}

func (s *chap7Handler) handleRoomConnection(roomID string, conn *websocket.Conn) {
	room := s.roomFactory.getOrCreate(roomID)
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
			s.sendMessage(room, conn, &InfoMessage{
				Uri:     "out/error",
				Message: "Message uri not recognized",
			})
			log.Println("No handler for message type: ", m.Uri)
		}
	}
}

func (s *chap7Handler) RoomWS(w http.ResponseWriter, r *http.Request) {
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

	s.handleRoomConnection(roomID, c)
}
