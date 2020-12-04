package chap7

import (
	"log"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

func CloneTrack(pc *webrtc.PeerConnection, targetTrack *webrtc.TrackLocalStaticRTP) func(t *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
	return func(t *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		go func() {
			ticker := time.NewTicker(3 * time.Second)
			for range ticker.C {
				//err := pc.WriteRTCP(
				//	[]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(t.SSRC())}},
				//)
				//if err != nil {
				//	log.Println("Error:: ", err.Error())
				//}

				err := pc.WriteRTCP(
					[]rtcp.Packet{&rtcp.ReceiverEstimatedMaximumBitrate{Bitrate: 10000000, SenderSSRC: uint32(t.SSRC())}},
				)
				if err != nil {
					log.Println("Error:: ", err.Error())
				}

			}
		}()

		for {
			// Read RTP packets being sent to Pion
			rtp, readErr := t.ReadRTP()
			if readErr != nil {
				panic(readErr)
			}
			if writeErr := targetTrack.WriteRTP(rtp); writeErr != nil {
				panic(writeErr)
			}
		}
	}
}
