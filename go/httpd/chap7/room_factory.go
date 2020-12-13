package chap7

import (
	"log"
	"sync"

	"github.com/andrefsp/video-democry/go/config"
)

// Room factory manages room creations
type roomFactory struct {
	cfg *config.Config

	roomsMutex sync.RWMutex
	rooms      map[string]*room
}

func (f *roomFactory) deleteIfEmpty(r *room) bool {
	f.roomsMutex.Lock()
	defer f.roomsMutex.Unlock()

	if len(r.getUserList()) < 1 {
		delete(f.rooms, r.ID)
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

		rooms: map[string]*room{},
	}

}
