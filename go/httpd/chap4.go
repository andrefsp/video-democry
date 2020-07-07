package httpd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type message struct {
	Uri string `json:"uri"`
}

type joinMessage struct {
	Participant *participant `json:"participant"`
}

type participant struct {
	Username string `json:"username"`
}

type room struct {
	participants map[*websocket.Conn]*participant
}

func (r *room) getParticipantList() []*participant {
	participants := []*participant{}
	for p := range r.participants {
		participants = append(participants, r.participants[p])
	}
	return participants
}

var rooms = map[string]*room{}

func (s *server) showParticipants() {
	roomParticipats := map[string][]*participant{}
	for roomID, room := range rooms {
		roomParticipats[roomID] = room.getParticipantList()
	}
	fmt.Println(roomParticipats)
}

func (s *server) handleUserJoined(r *room, conn *websocket.Conn, messagePayload []byte) error {
	uj := joinMessage{}
	if err := json.Unmarshal(messagePayload, &uj); err != nil {
		return err
	}

	r.participants[conn] = uj.Participant
	s.pushParticipants(r)
	return nil
}

func (s *server) pushParticipants(r *room) error {
	// send current users on the call
	jData, err := json.Marshal(r.getParticipantList())
	if err != nil {
		return err
	}

	for conn := range r.participants {
		if err := conn.WriteMessage(websocket.TextMessage, jData); err != nil {
			log.Println("write err:", err)
			return err
		}
	}
	return nil
}

func (s *server) handleConnection(roomID string, conn *websocket.Conn) {
	var r *room
	r, ok := rooms[roomID]
	if !ok {
		log.Printf("Created room %s \n", roomID)
		r = &room{
			participants: map[*websocket.Conn]*participant{},
		}
		rooms[roomID] = r
	}

	defer func(c *websocket.Conn, rr *room) {
		log.Printf("Connection went away %s \n")
		delete(r.participants, c)
		c.Close()

		s.pushParticipants(rr)
	}(conn, r)

	for {
		_, messagePayload, err := conn.ReadMessage()
		if err != nil {
			log.Println("read err:", err)
			break
		}

		m := message{}
		if err := json.Unmarshal(messagePayload, &m); err != nil {
			log.Println("read err:", err)
			break
		}

		switch m.Uri {
		case "join":
			s.handleUserJoined(r, conn, messagePayload)
		default:
			log.Println("No handler for message type: ", m.Uri)
		}

	}
}

func (s *server) chap4(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		response(w, http.StatusBadRequest, newError("room not present on request"))
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		response(w, http.StatusBadRequest, newError(err.Error()))
		return
	}

	go s.handleConnection(roomID, c)
}
