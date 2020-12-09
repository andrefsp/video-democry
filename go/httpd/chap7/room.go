package chap7

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type subscriberTracks struct {
	audioTrack *webrtc.TrackLocalStaticRTP
	videoTrack *webrtc.TrackLocalStaticRTP
}

// models
type user struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	StreamID string `json:"streamID"`

	subscribersMutex sync.RWMutex
	subscribers      map[string]*subscriberTracks

	pc    *webrtc.PeerConnection
	audio *webrtc.TrackRemote
	video *webrtc.TrackRemote

	startVideoBrodcast chan struct{}
	startAudioBrodcast chan struct{}
}

func (u *user) addVideoTrack(video *webrtc.TrackRemote) {
	u.video = video
	u.startVideoBrodcast <- struct{}{}
}

func (u *user) addAudioTrack(audio *webrtc.TrackRemote) {
	u.audio = audio
	u.startAudioBrodcast <- struct{}{}
}

func (u *user) broadcastAudio() {
	<-u.startAudioBrodcast
	for {
		// Read RTP packets being sent to Pion
		rtp, readErr := u.audio.ReadRTP()
		if readErr != nil {
			panic(readErr)
		}

		for id := range u.subscribers {
			if writeErr := u.subscribers[id].audioTrack.WriteRTP(rtp); writeErr != nil {
				panic(writeErr)
			}
		}
	}
}

func (u *user) broadcastVideo() {
	<-u.startVideoBrodcast
	for {
		// Read RTP packets being sent to Pion
		rtp, readErr := u.video.ReadRTP()
		if readErr != nil {
			panic(readErr)
		}
		for id := range u.subscribers {
			if writeErr := u.subscribers[id].videoTrack.WriteRTP(rtp); writeErr != nil {
				panic(writeErr)
			}
		}
	}
}

func (u *user) addSubscriber(subscriber *user) error {

	if u.hasSubscriber(subscriber) {
		return nil
	}

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "video/vp8"},
		"video",
		u.StreamID,
	)
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return err
	}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "audio/opus"},
		"audio",
		u.StreamID,
	)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}

	u.subscribersMutex.Lock()
	u.subscribers[subscriber.ID] = &subscriberTracks{
		audioTrack: audioTrack,
		videoTrack: videoTrack,
	}
	u.subscribersMutex.Unlock()

	// Must add the tracks to the subscriber
	subscriber.pc.AddTrack(audioTrack)
	subscriber.pc.AddTrack(videoTrack)

	return nil
}

func (u *user) hasSubscriber(subscriber *user) bool {
	u.subscribersMutex.RLock()
	defer u.subscribersMutex.RUnlock()
	_, subscribed := u.subscribers[subscriber.ID]

	return subscribed
}

func newUser(u *user) *user {
	newUser := &user{
		ID:       u.ID,
		Username: u.Username,
		StreamID: u.StreamID,

		subscribersMutex: sync.RWMutex{},
		subscribers:      map[string]*subscriberTracks{},

		startVideoBrodcast: make(chan struct{}),
		startAudioBrodcast: make(chan struct{}),
	}

	go newUser.broadcastAudio()
	go newUser.broadcastVideo()

	return newUser
}

type room struct {
	users map[*websocket.Conn]*user

	ticker   <-chan time.Time
	stopChan chan struct{}
}

func (r *room) subscribeTracks(publisher *user, subscriber *user) error {
	if publisher.ID == subscriber.ID {
		return nil
	}

	if publisher.hasSubscriber(subscriber) {
		log.Println("User already subscribed")
		return nil
	}

	if publisher.audio == nil || publisher.video == nil {
		log.Printf("`%s` user is not yet streaming.", publisher.ID)
		return nil
	}

	if err := publisher.addSubscriber(subscriber); err != nil {
		log.Print("Error: ", err.Error())
		return err
	}

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
