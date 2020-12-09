package chap7

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

type subscriberTracks struct {
	audioTrack *webrtc.TrackLocalStaticRTP
	videoTrack *webrtc.TrackLocalStaticRTP

	videoRTPSender *webrtc.RTPSender
	audioRTPSender *webrtc.RTPSender
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

	stopped bool
}

func (u *user) stop() {
	u.stopped = true
	u.pc.Close()
}

func (u *user) sendRemb(t *webrtc.TrackRemote) {
	ticker := time.NewTicker(3 * time.Second)
	for range ticker.C {
		if u.stopped || u.pc.ConnectionState() != webrtc.PeerConnectionStateConnected {
			return
		}

		writeErr := u.pc.WriteRTCP([]rtcp.Packet{
			&rtcp.PictureLossIndication{
				MediaSSRC: uint32(t.SSRC()),
			},
		})
		if writeErr != nil {
			log.Println(writeErr)
		}
		// Send a remb message with a very high bandwidth to trigger chrome to send also the high bitrate stream
		writeErr = u.pc.WriteRTCP([]rtcp.Packet{
			&rtcp.ReceiverEstimatedMaximumBitrate{
				Bitrate:    10000000,
				SenderSSRC: uint32(t.SSRC()),
			}})
		if writeErr != nil {
			log.Println(writeErr)
		}
	}
}

func (u *user) addVideoTrack(video *webrtc.TrackRemote) {
	go u.sendRemb(video)

	u.video = video
	u.startVideoBrodcast <- struct{}{}
}

func (u *user) addAudioTrack(audio *webrtc.TrackRemote) {
	go u.sendRemb(audio)

	u.audio = audio
	u.startAudioBrodcast <- struct{}{}
}

func (u *user) broadcastAudio() {
	<-u.startAudioBrodcast
	for {
		if u.stopped {
			return
		}
		// Read RTP packets being sent to Pion
		rtp, err := u.audio.ReadRTP()
		if err != nil {
			log.Printf("Error: %s\n", err.Error())
			return
		}

		u.subscribersMutex.RLock()
		for id := range u.subscribers {
			if writeErr := u.subscribers[id].audioTrack.WriteRTP(rtp); writeErr != nil {
				panic(writeErr)
			}
		}
		u.subscribersMutex.RUnlock()
	}
}

func (u *user) broadcastVideo() {
	<-u.startVideoBrodcast
	for {
		if u.stopped {
			return
		}

		// Read RTP packets being sent to Pion
		rtp, err := u.video.ReadRTP()
		if err != nil {
			log.Printf("Error: %s\n", err.Error())
			return
		}

		u.subscribersMutex.RLock()
		for id := range u.subscribers {
			if writeErr := u.subscribers[id].videoTrack.WriteRTP(rtp); writeErr != nil {
				panic(writeErr)
			}
		}
		u.subscribersMutex.RUnlock()
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

	// Must add the tracks to the subscriber
	audioRTPSender, err := subscriber.pc.AddTrack(audioTrack)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}

	videoRTPSender, err := subscriber.pc.AddTrack(videoTrack)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}

	u.subscribersMutex.Lock()
	u.subscribers[subscriber.ID] = &subscriberTracks{
		audioTrack: audioTrack,
		videoTrack: videoTrack,

		audioRTPSender: audioRTPSender,
		videoRTPSender: videoRTPSender,
	}
	u.subscribersMutex.Unlock()

	return nil
}

func (u *user) removeSubscriber(subscriber *user) error {
	if !u.hasSubscriber(subscriber) {
		// Can only remove if subscribed
		return nil
	}

	u.subscribersMutex.Lock()
	defer u.subscribersMutex.Unlock()

	theirs := u.subscribers[subscriber.ID]
	if err := subscriber.pc.RemoveTrack(theirs.audioRTPSender); err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}

	if err := subscriber.pc.RemoveTrack(theirs.videoRTPSender); err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}

	mytracks := subscriber.subscribers[u.ID]
	if err := u.pc.RemoveTrack(mytracks.audioRTPSender); err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}
	if err := u.pc.RemoveTrack(mytracks.videoRTPSender); err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err

	}

	delete(u.subscribers, subscriber.ID)

	return nil
}

func (u *user) hasSubscriber(subscriber *user) bool {
	u.subscribersMutex.RLock()
	defer u.subscribersMutex.RUnlock()
	_, subscribed := u.subscribers[subscriber.ID]

	return subscribed
}

func (u *user) showSubscribers() {
	ticker := time.NewTicker(5 * time.Second)

	for range ticker.C {
		if u.stopped {
			return
		}

		log.Printf("User: %s, subscribers: %d, senders: %d", u.ID, len(u.subscribers), len(u.pc.GetSenders()))
	}
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

		stopped: false,
	}

	go newUser.broadcastAudio()
	go newUser.broadcastVideo()

	go newUser.showSubscribers()

	return newUser
}

type room struct {
	users map[*websocket.Conn]*user

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
	return r.users[conn]
}

func (r *room) addUser(conn *websocket.Conn, user *user) *user {
	r.users[conn] = user
	return user
}

func (r *room) removeUser(conn *websocket.Conn, user *user) *user {
	// Must unsubscribe from tracks.
	for _, publisher := range r.getUserList() {
		if publisher.ID == user.ID {
			continue
		}
		if err := publisher.removeSubscriber(user); err != nil {
			log.Print("Error: ", err.Error())
		}
	}

	user = r.users[conn]
	user.stop()

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
