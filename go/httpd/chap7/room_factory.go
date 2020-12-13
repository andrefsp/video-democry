package chap7

import (
	"log"
	"sync"

	"github.com/andrefsp/video-democry/go/config"
	"github.com/gorilla/websocket"
)

type eventSubscription struct {
	roomCreated chan *room
	roomDeleted chan *room
}

// Room factory manages room creations
type roomFactory struct {
	cfg *config.Config

	roomsMutex sync.RWMutex
	rooms      map[string]*room

	eventSubscriptionsMutext sync.Mutex
	eventSubscriptions       map[*websocket.Conn]*eventSubscription
}

func (f *roomFactory) notify(r *room, action string) {
	for _, subscription := range f.eventSubscriptions {
		switch action {
		case "deleted":
			subscription.roomDeleted <- r
		case "created":
			subscription.roomCreated <- r
		}
	}
}

func (f *roomFactory) subscribe(conn *websocket.Conn) *eventSubscription {
	f.eventSubscriptionsMutext.Lock()
	defer f.eventSubscriptionsMutext.Unlock()

	f.eventSubscriptions[conn] = &eventSubscription{
		roomCreated: make(chan *room),
		roomDeleted: make(chan *room),
	}

	return f.eventSubscriptions[conn]
}

func (f *roomFactory) deleteIfEmpty(r *room) bool {
	f.roomsMutex.Lock()
	defer f.roomsMutex.Unlock()

	if len(r.getUserList()) < 1 {
		delete(f.rooms, r.ID)
		defer f.notify(f.rooms[r.ID], "deleted")
		return true
	}

	return false
}

func (f *roomFactory) getOrCreate(id string) *room {
	f.roomsMutex.Lock()
	defer f.roomsMutex.Unlock()

	if r, ok := f.rooms[id]; ok {
		return r
	}

	defer log.Printf("New room created. ID: `%s`", id)

	f.rooms[id] = newRoom(id)
	f.rooms[id].start()

	defer f.notify(f.rooms[id], "created")

	return f.rooms[id]
}

func (f *roomFactory) listRooms() []*room {
	f.roomsMutex.RLock()
	defer f.roomsMutex.RUnlock()

	roomsList := make([]*room, len(f.rooms))

	i := 0
	for _, room := range f.rooms {
		roomsList[i] = room
		i += 1
	}

	return roomsList
}

func newRoomFactory(cfg *config.Config) *roomFactory {
	return &roomFactory{
		cfg: cfg,

		rooms:              map[string]*room{},
		eventSubscriptions: map[*websocket.Conn]*eventSubscription{},
	}

}
