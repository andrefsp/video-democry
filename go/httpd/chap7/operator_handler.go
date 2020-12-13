package chap7

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/andrefsp/video-democry/go/httpd/responses"
)

func (s *chap7Handler) handleOperatorConnection(conn *websocket.Conn) {
	for {
		_, messagePayload, err := conn.ReadMessage()
		if err != nil {
			log.Println("read err:", err)
			//s.handleDisconnection(room, conn)
			break
		}
		log.Println(string(messagePayload))
	}
}

func (s *chap7Handler) OperatorWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		responses.Send(w, http.StatusBadRequest, responses.NewError(err.Error()))
		return
	}
	s.sendMessage(nil, conn, s.roomFactory.listRooms())

	s.handleOperatorConnection(conn)
}
