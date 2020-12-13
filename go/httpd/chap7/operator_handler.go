package chap7

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/andrefsp/video-democry/go/httpd/responses"
)

func (s *chap7Handler) handleOperatorDisconnection(conn *websocket.Conn, dchan chan struct{}) {
	dchan <- struct{}{}
}

func (s *chap7Handler) handleOperatorConnection(conn *websocket.Conn) {

	messageChan := make(chan []byte)
	disconnectChan := make(chan struct{})

	go func(mchan chan []byte, dchan chan struct{}) {
		for {
			_, messagePayload, err := conn.ReadMessage()
			if err != nil {
				log.Println("read err:", err)
				s.handleOperatorDisconnection(conn, dchan)
				break
			}
			mchan <- messagePayload
		}
	}(messageChan, disconnectChan)

	roomFactoryEvents := s.roomFactory.subscribe(conn)

	// send current list of rooms
	s.sendMessage(nil, conn, s.roomFactory.listRooms())

	for {
		select {
		case payload := <-messageChan:
			log.Println(string(payload))
		case <-roomFactoryEvents.roomCreated:
			s.sendMessage(nil, conn, s.roomFactory.listRooms())
		case <-roomFactoryEvents.roomDeleted:
			s.sendMessage(nil, conn, s.roomFactory.listRooms())
		case <-disconnectChan:
			return
		}
	}

}

func (s *chap7Handler) OperatorWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		responses.Send(w, http.StatusBadRequest, responses.NewError(err.Error()))
		return
	}

	s.handleOperatorConnection(conn)
}
