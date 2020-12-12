package chap7

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const MaxRoomSize = 2

var ErrMaxUsersPerRoom = errors.New("Maximum users in room")

type room struct {
	messageMutex sync.Mutex

	usersMutex sync.RWMutex
	users      map[*websocket.Conn]*user

	ticker   <-chan time.Time
	stopChan chan struct{}
}

func (r *room) handleStreamSubscriptions() error {
	for _, publisher := range r.getUserList() {
		for _, subscriber := range r.getUserList() {
			if publisher.ID == subscriber.ID {
				continue
			}

			if err := publisher.addSubscriber(subscriber); err != nil {
				log.Print("Error: ", err.Error())
				return err
			}
		}
	}
	return nil
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
	r.usersMutex.RLock()
	defer r.usersMutex.RUnlock()

	return r.users[conn]
}

func (r *room) addUser(conn *websocket.Conn, user *user) (*user, error) {
	r.usersMutex.Lock()
	defer r.usersMutex.Unlock()

	if len(r.users) >= MaxRoomSize {
		return nil, ErrMaxUsersPerRoom
	}

	r.users[conn] = user
	return user, nil
}

func (r *room) removeUser(conn *websocket.Conn) *user {
	user := r.getUser(conn)
	if user == nil {
		return user
	}

	// Must unsubscribe from tracks.
	for _, publisher := range r.getUserList() {
		if publisher.ID == user.ID {
			continue
		}
		if err := publisher.removeSubscriber(user); err != nil {
			log.Print("Error: ", err.Error())
		}
		if err := user.removeSubscriber(publisher); err != nil {
			log.Print("Error: ", err.Error())
		}
	}

	r.usersMutex.Lock()
	defer r.usersMutex.Unlock()

	user = r.users[conn]
	user.stop()

	delete(r.users, conn)

	return user
}

func (r *room) getUserList() []*user {
	r.usersMutex.RLock()
	defer r.usersMutex.RUnlock()

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
