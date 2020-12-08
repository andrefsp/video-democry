package chap7

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// models
type user struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	StreamID string `json:"streamID"`

	subscribers map[string]struct{}

	pc    *webrtc.PeerConnection
	audio *webrtc.TrackRemote
	video *webrtc.TrackRemote
}

type room struct {
	users map[*websocket.Conn]*user

	subscriptionMutex sync.Mutex

	ticker   <-chan time.Time
	stopChan chan struct{}
}

func (r *room) subscribeTracks(publisher *user, subscriber *user) error {
	r.subscriptionMutex.Lock()
	defer r.subscriptionMutex.Unlock()

	if _, subscribed := publisher.subscribers[subscriber.ID]; subscribed {
		log.Println("User already subscribed")
		return nil
	}

	if publisher.ID == subscriber.ID {
		log.Println("Cannot subscribe to self.")
		return nil
	}

	if publisher.audio == nil || publisher.video == nil {
		log.Printf("`%s` user is not yet streaming.", publisher.ID)
		return nil
	}

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "video/vp8"},
		"video",
		publisher.StreamID,
	)
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return err
	}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "audio/opus"},
		"audio",
		publisher.StreamID,
	)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}

	writeRTP := func(sourceTrack *webrtc.TrackRemote, targetTrack *webrtc.TrackLocalStaticRTP) {
		for {
			// Read RTP packets being sent to Pion
			rtp, readErr := sourceTrack.ReadRTP()
			if readErr != nil {
				panic(readErr)
			}
			if writeErr := targetTrack.WriteRTP(rtp); writeErr != nil {
				panic(writeErr)
			}
		}
	}

	go writeRTP(publisher.audio, audioTrack)
	go writeRTP(publisher.video, videoTrack)

	if _, err = subscriber.pc.AddTrack(videoTrack); err != nil {
		return err
	}
	if _, err = subscriber.pc.AddTrack(audioTrack); err != nil {
		return err
	}

	publisher.subscribers[subscriber.ID] = struct{}{}

	return nil
}

func (r *room) handleStreamSubscriptions() {
	for _, publisher := range r.getUserList() {
		for _, subscriber := range r.getUserList() {
			if publisher.ID == subscriber.ID {
				continue
			}
			r.subscribeTracks(publisher, subscriber)
		}
	}
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
