package chap7

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// models
type user struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	StreamID string `json:"streamID"`

	pc    *webrtc.PeerConnection
	audio *webrtc.TrackRemote
	video *webrtc.TrackRemote
}

type room struct {
	users map[*websocket.Conn]*user

	ticker   <-chan time.Time
	stopChan chan struct{}
}

func (r *room) stop() {
	r.stopChan <- struct{}{}
}

func (r *room) start() {
	go func() {
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
	}()
}

func (r *room) getUser(conn *websocket.Conn) *user {
	return r.users[conn]
}

func (r *room) addUser(conn *websocket.Conn, user *user) *user {
	r.users[conn] = user
	return user
}

func (r *room) removeUser(conn *websocket.Conn, user *user) *user {
	user = r.users[conn]
	delete(r.users, conn)
	return user
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
	return r
}
