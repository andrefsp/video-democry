package chap7

import (
	"log"
	//	"sync"
	//	"time"

	//	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

func (s *chap7Handler) subscribeTracks(tuser *user, subscriber *user) {
	if tuser.audio == nil || tuser.video == nil {
		log.Println("Target user is not yet streaming.")
		return
	}

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "video/vp8"},
		"video",
		tuser.StreamID,
	)
	if err != nil {
		log.Printf("Error: %s", err.Error())
		panic(err)
	}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "audio/opus"},
		"audio",
		tuser.StreamID,
	)
	if err != nil {
		log.Printf("Error: %s", err.Error())
		panic(err)
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

	go writeRTP(tuser.audio, audioTrack)
	go writeRTP(tuser.video, videoTrack)

	subscriber.pc.AddTrack(videoTrack)
	subscriber.pc.AddTrack(audioTrack)
}
