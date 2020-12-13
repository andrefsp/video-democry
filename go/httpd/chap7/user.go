package chap7

import (
	"log"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"

	"github.com/andrefsp/video-democry/go/config"
)

type subscriberRTPSenders struct {
	videoRTPSender *webrtc.RTPSender
	audioRTPSender *webrtc.RTPSender
}

// models
type user struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	StreamID string `json:"streamID"`

	subscribersMutex sync.RWMutex
	subscribers      map[string]*subscriberRTPSenders

	pc *webrtc.PeerConnection

	audioInTrack *webrtc.TrackRemote
	videoInTrack *webrtc.TrackRemote

	audioOutTrack *webrtc.TrackLocalStaticRTP
	videoOutTrack *webrtc.TrackLocalStaticRTP

	startVideoBrodcast chan struct{}
	startAudioBrodcast chan struct{}

	stopped bool
}

func (u *user) stop() {
	log.Println("User:: ", u)
	u.stopped = true
	if u.pc != nil {
		u.pc.Close()
	}
}

func (u *user) sendPLI(t *webrtc.TrackRemote) {
	ticker := time.NewTicker(3 * time.Second)
	for range ticker.C {
		if u.stopped {
			return
		}
		if u.pc.ConnectionState() != webrtc.PeerConnectionStateConnected {
			continue
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

func (u *user) addVideoTrack(video *webrtc.TrackRemote) error {
	go u.sendPLI(video)

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{
			MimeType: video.Codec().MimeType,
		},
		"video",
		u.StreamID,
	)
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return err
	}

	u.videoOutTrack = videoTrack
	u.videoInTrack = video
	u.startVideoBrodcast <- struct{}{}

	return nil
}

func (u *user) addAudioTrack(audio *webrtc.TrackRemote) error {
	go u.sendPLI(audio)

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{
			MimeType: audio.Codec().MimeType,
		},
		"audio",
		u.StreamID,
	)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}

	u.audioOutTrack = audioTrack
	u.audioInTrack = audio
	u.startAudioBrodcast <- struct{}{}

	return nil
}

func (u *user) broadcastAudio() {
	<-u.startAudioBrodcast
	for {
		if u.stopped {
			return
		}
		// Read RTP packets being sent to Pion
		rtp, err := u.audioInTrack.ReadRTP()
		if err != nil {
			log.Printf("Error broadcasting audio: %s\n", err.Error())
			return
		}

		if writeErr := u.audioOutTrack.WriteRTP(rtp); writeErr != nil {
			panic(writeErr)
		}
	}
}

func (u *user) broadcastVideo() {
	<-u.startVideoBrodcast
	for {
		if u.stopped {
			return
		}

		// Read RTP packets being sent to Pion
		rtp, err := u.videoInTrack.ReadRTP()
		if err != nil {
			log.Printf("Error broadcasting video: %s\n", err.Error())
			return
		}
		if writeErr := u.videoOutTrack.WriteRTP(rtp); writeErr != nil {
			panic(writeErr)
		}
	}
}

func (u *user) addSubscriber(subscriber *user) error {
	defer log.Printf("`%s` subscribed to `%s`", subscriber.ID, u.ID)

	u.subscribersMutex.Lock()
	defer u.subscribersMutex.Unlock()

	for {
		// Wait until tracks are set.
		if u.videoOutTrack != nil && u.audioOutTrack != nil {
			break
		}
		time.Sleep(time.Second)
	}

	if _, subscribed := u.subscribers[subscriber.ID]; subscribed {
		// Return if already subscribed
		return nil
	}

	// Must add the tracks to the subscriber
	audioRTPSender, err := subscriber.pc.AddTrack(u.audioOutTrack)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}

	videoRTPSender, err := subscriber.pc.AddTrack(u.videoOutTrack)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}
	u.subscribers[subscriber.ID] = &subscriberRTPSenders{
		audioRTPSender: audioRTPSender,
		videoRTPSender: videoRTPSender,
	}

	return nil
}

func (u *user) removeSubscriber(subscriber *user) error {
	defer log.Printf("`%s` unsubscribed to `%s`", subscriber.ID, u.ID)

	u.subscribersMutex.Lock()
	defer u.subscribersMutex.Unlock()

	if _, subscribed := u.subscribers[subscriber.ID]; !subscribed {
		// Can only remove if subscribed
		return nil
	}
	theirs := u.subscribers[subscriber.ID]
	if err := subscriber.pc.RemoveTrack(theirs.audioRTPSender); err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}

	if err := subscriber.pc.RemoveTrack(theirs.videoRTPSender); err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}

	delete(u.subscribers, subscriber.ID)

	return nil
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

func (s *userFactory) newPeerConnection() (*webrtc.PeerConnection, error) {
	me, err := getPublisherMediaEngine()
	if err != nil {
		return nil, err
	}

	return webrtc.NewAPI(webrtc.WithMediaEngine(me)).
		//return webrtc.
		NewPeerConnection(webrtc.Configuration{
			SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
			//ICETransportPolicy: webrtc.ICETransportPolicyRelay,
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
				{
					URLs:       []string{s.cfg.TurnServerAddr},
					Credential: "thiskey",
					Username:   "thisuser",
				},
			},
		})
}

type userFactory struct {
	cfg *config.Config
}

func (f *userFactory) newUser(u *user) (*user, error) {
	pc, err := f.newPeerConnection()
	if err != nil {
		log.Print("Error creating Peer connection: ", err.Error())
		return nil, err
	}

	newUser := &user{
		ID:       u.ID,
		Username: u.Username,
		StreamID: u.StreamID,

		pc:               pc,
		subscribersMutex: sync.RWMutex{},
		subscribers:      map[string]*subscriberRTPSenders{},

		startVideoBrodcast: make(chan struct{}),
		startAudioBrodcast: make(chan struct{}),

		stopped: false,
	}

	go newUser.broadcastAudio()
	go newUser.broadcastVideo()

	// go newUser.showSubscribers()

	return newUser, nil
}

func newUserFactory(cfg *config.Config) *userFactory {
	return &userFactory{
		cfg: cfg,
	}
}
