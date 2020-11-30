package chap7

import (
	"time"

	"github.com/gorilla/websocket"
)

type room struct {
	users map[*websocket.Conn]*user

	ticker   <-chan time.Time
	stopChan chan struct{}
}

func (r *room) stop() {
	r.stopChan <- struct{}{}
}

func (r *room) start() {
	for {
		select {
		case <-r.ticker:
			for conn := range r.users {
				conn.WriteMessage(websocket.TextMessage, []byte(`{"uri":"out/ping"}`))
			}
		case <-r.stopChan:
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
		users:    map[*websocket.Conn]*user{},
		ticker:   time.NewTicker(15 * time.Second).C,
		stopChan: make(chan struct{}, 1),
	}
	go r.start()
	return r
}
