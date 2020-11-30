package chap6

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/andrefsp/video-democry/go/config"
	"github.com/andrefsp/video-democry/go/httpd/responses"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	webrtc "github.com/pion/webrtc/v3"

	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/oggreader"
)

var videoFileName = "/home/andrefsp/development/video-democry/src/github.com/andrefsp/video-democry/go/httpd/chap6/output.ivf"
var audioFileName = "/home/andrefsp/development/video-democry/src/github.com/andrefsp/video-democry/go/httpd/chap6/output.ogg"

func getPayloadType(m webrtc.MediaEngine, codecType webrtc.RTPCodecType, codecName string) (uint8, error) {
	for _, codec := range m.GetCodecsByKind(codecType) {
		if codec.Name == codecName {
			return codec.PayloadType, nil
		}
	}
	return 0, errors.New(fmt.Sprintf("Remote peer does not support %s", codecName))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// messages
type InfoMessage struct {
	Uri     string `json:"uri"`
	Message string `json:"message"`
}

type message struct {
	Uri string `json:"uri"`
}

type InICECandidate struct {
	FromUser  *user                   `json:"from_user"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type OutICECandidate struct {
	Uri       string                  `json:"uri"`
	ToUser    *user                   `json:"to_user"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type InOffer struct {
	FromUser *user                     `json:"from_user"`
	Offer    webrtc.SessionDescription `json:"offer"`
}

type OutAnswer struct {
	Uri    string                    `json:"uri"`
	ToUser *user                     `json:"to_user"`
	Answer webrtc.SessionDescription `json:"answer"`
}

type InUserJoinMessage struct {
	User *user `json:"user"`
}

type OutRoomEventMessage struct {
	Uri   string  `json:"uri"`
	User  *user   `json:"user"`
	Users []*user `json:"room_users"`
}

// models
type user struct {
	Username string `json:"username"`
	StreamID string `json:"stream_id"`

	pc *webrtc.PeerConnection
}

type room struct {
	users map[*websocket.Conn]*user

	ticker <-chan time.Time
	stop   <-chan struct{}
}

func (r *room) start() {
	for {
		select {
		case <-r.ticker:
			for conn := range r.users {
				conn.WriteMessage(websocket.TextMessage, []byte(`{"uri":"out/ping"}`))
			}
		case <-r.stop:
			break
		}
	}
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
		users:  map[*websocket.Conn]*user{},
		ticker: time.NewTicker(15 * time.Second).C,
		stop:   make(<-chan struct{}, 1),
	}
	go r.start()
	return r
}

var rooms = map[string]*room{}

type chap6Handler struct {
	cfg *config.Config
}

func (s *chap6Handler) sendAudio(me webrtc.MediaEngine, peerConnection *webrtc.PeerConnection, wait <-chan struct{}) error {

	codec, err := getPayloadType(me, webrtc.RTPCodecTypeAudio, "opus")
	if err != nil {
		return err
	}

	audioTrack, err := peerConnection.NewTrack(codec, rand.Uint32(), "audio", "pion")
	if err != nil {
		return err
	}
	if _, err = peerConnection.AddTrack(audioTrack); err != nil {
		return err
	}

	go func() {
		file, err := os.Open(audioFileName)
		if err != nil {
			panic(err)
		}
		ogg, _, err := oggreader.NewWith(file)
		if err != nil {
			panic(err)
		}
		<-wait
		log.Printf("sending audio.")
		var lastGranule uint64
		for {
			pageData, pageHeader, err := ogg.ParseNextPage()
			if err == io.EOF {
				return
			}
			if err != nil {
				panic(err)
			}

			sampleCount := float64(pageHeader.GranulePosition - lastGranule)
			lastGranule = pageHeader.GranulePosition

			err = audioTrack.WriteSample(media.Sample{Data: pageData, Samples: uint32(sampleCount)})
			if err != nil {
				panic(err)
			}

			time.Sleep(time.Duration((sampleCount/48000)*1000) * time.Millisecond)
		}

	}()

	return nil
}

func (s *chap6Handler) sendVideo(me webrtc.MediaEngine, peerConnection *webrtc.PeerConnection, wait <-chan struct{}) error {

	codec, err := getPayloadType(me, webrtc.RTPCodecTypeVideo, "VP8")
	if err != nil {
		return err
	}

	track, err := peerConnection.NewTrack(codec, rand.Uint32(), "video", "pion")
	if err != nil {
		return err
	}

	if _, err := peerConnection.AddTrack(track); err != nil {
		return err
	}

	go func() {
		file, err := os.Open(videoFileName)
		if err != nil {
			panic(err)
		}

		ivf, header, err := ivfreader.NewWith(file)
		if err != nil {
			panic(err)
		}

		<-wait

		log.Println("Sending video...")
		sleepTime := time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000)
		for {
			frame, _, err := ivf.ParseNextFrame()
			if err == io.EOF {
				fmt.Printf("All audio pages parsed and sent")
				return
			}

			if err != nil {
				panic(err)
			}

			// The amount of samples is the difference between the last and current timestamp
			time.Sleep(sleepTime)
			if err = track.WriteSample(media.Sample{Data: frame, Samples: 90000}); err != nil {
				panic(err)
			}

		}

	}()

	return nil
}

func (s *chap6Handler) sendMessage(conn *websocket.Conn, payload interface{}) error {
	jData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, jData)
}

func (s *chap6Handler) handleUserJoined(r *room, conn *websocket.Conn, messagePayload []byte) error {
	uj := InUserJoinMessage{}
	if err := json.Unmarshal(messagePayload, &uj); err != nil {
		return err
	}
	if _, ok := r.users[conn]; ok {
		s.sendMessage(conn, &InfoMessage{Uri: "out/info", Message: "User already joined"})
		return nil
	}
	r.users[conn] = uj.User
	s.pushRoomStatus(r, uj.User, "out/user-join")
	return nil
}

func (s *chap6Handler) handleICECandidate(r *room, conn *websocket.Conn, messagePayload []byte) error {
	cm := InICECandidate{}
	if err := json.Unmarshal(messagePayload, &cm); err != nil {
		return err
	}

	if cm.Candidate.Candidate == "" {
		return nil
	}

	if err := r.users[conn].pc.AddICECandidate(cm.Candidate); err != nil {
		log.Printf("Error adding ICECandidate(%s): (%+v)\n", err.Error(), cm.Candidate)
		return err
	}
	log.Println("Added ICECandidate")
	return nil
}

func (s *chap6Handler) handleOffer(r *room, conn *websocket.Conn, messagePayload []byte) error {
	om := InOffer{}
	if err := json.Unmarshal(messagePayload, &om); err != nil {
		return err
	}

	mediaEngine := webrtc.MediaEngine{}
	if err := mediaEngine.PopulateFromSDP(om.Offer); err != nil {
		return err
	}

	pc, err := webrtc.
		NewAPI(webrtc.WithMediaEngine(mediaEngine)).
		NewPeerConnection(webrtc.Configuration{
			ICETransportPolicy: webrtc.ICETransportPolicyRelay,
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
				{
					URLs:       []string{"turn:192.168.0.39:3478"},
					Credential: "thiskey",
					Username:   "thisuser",
				},
			},
		})
	if err != nil {
		log.Print("Error creating Peer connection: ", err.Error())
		return err
	}

	r.users[conn].pc = pc

	startVideo := make(chan struct{}, 1)
	startAudio := make(chan struct{}, 1)

	if err := s.sendVideo(mediaEngine, r.users[conn].pc, startVideo); err != nil {
		log.Print("Error sending video: ", err.Error())
		return err
	}

	if err := s.sendAudio(mediaEngine, r.users[conn].pc, startAudio); err != nil {
		log.Print("Error sending video: ", err.Error())
		return err
	}

	if err := r.users[conn].pc.SetRemoteDescription(om.Offer); err != nil {
		log.Print("Error: ", err.Error())
		return err
	}

	answer, err := r.users[conn].pc.CreateAnswer(nil)
	if err != nil {
		log.Print("Error: ", err.Error())
		return err
	}

	if err = r.users[conn].pc.SetLocalDescription(answer); err != nil {
		log.Print("Error: ", err.Error())
		return err
	}

	r.users[conn].pc.OnICEConnectionStateChange(func(c webrtc.ICEConnectionState) {
		if c == webrtc.ICEConnectionStateConnected {
			log.Println("Connected...")
			startVideo <- struct{}{}
			startAudio <- struct{}{}
		}
	})

	r.users[conn].pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		s.sendMessage(conn, &OutICECandidate{
			Uri:       "out/icecandidate",
			ToUser:    r.users[conn],
			Candidate: c.ToJSON(),
		})
	})

	s.sendMessage(conn, &OutAnswer{
		Uri:    "out/answer",
		Answer: answer,
		ToUser: om.FromUser, // We are answering to the same user.
	})

	return nil
}

func (s *chap6Handler) handleAnswer(r *room, conn *websocket.Conn, messagePayload []byte) error {
	return nil
}

func (s *chap6Handler) pushRoomStatus(r *room, u *user, uri string) error {
	// send current users on the call

	payload := &OutRoomEventMessage{
		Uri:   uri,
		User:  u,
		Users: r.getUserList(),
	}

	for conn := range r.users {
		if err := s.sendMessage(conn, payload); err != nil {
			log.Println("write err:", err)
			return err
		}
	}
	return nil
}

func (s *chap6Handler) handleDisconnection(r *room, conn *websocket.Conn) {
	defer conn.Close()
	log.Print("Connection went away...")

	u, ok := r.users[conn]
	if !ok {
		return
	}
	delete(r.users, conn)
	s.pushRoomStatus(r, u, "out/user-left")
}

func (s *chap6Handler) handleConnection(roomID string, conn *websocket.Conn) {
	var r *room
	r, ok := rooms[roomID]
	if !ok {
		log.Printf("Created room %s \n", roomID)
		r = newRoom()

		rooms[roomID] = r
	}

	for {
		_, messagePayload, err := conn.ReadMessage()
		if err != nil {
			log.Println("read err:", err)
			s.handleDisconnection(r, conn)
			break
		}

		m := message{}
		if err := json.Unmarshal(messagePayload, &m); err != nil {
			log.Println("read err:", err)
			continue
		}

		switch m.Uri {
		case "in/join":
			s.handleUserJoined(r, conn, messagePayload)
		case "in/icecandidate":
			s.handleICECandidate(r, conn, messagePayload)
		case "in/offer":
			s.handleOffer(r, conn, messagePayload)
		case "in/answer":
			s.handleAnswer(r, conn, messagePayload)
		case "in/pong":
		default:
			s.sendMessage(conn, &InfoMessage{
				Uri: "out/error", Message: "Message uri not recognized",
			})

			log.Println("No handler for message type: ", m.Uri)
		}

	}
}

func (s *chap6Handler) Handler(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		responses.Send(w, http.StatusBadRequest, responses.NewError("room not present on request"))
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		responses.Send(w, http.StatusBadRequest, responses.NewError(err.Error()))
		return
	}

	go s.handleConnection(roomID, c)
}

func (s *chap6Handler) RegisterHandlers(m *mux.Router, middleware func(h http.HandlerFunc) http.HandlerFunc) {
	m.HandleFunc("/endpoint", s.Handler)
}

func New(cfg *config.Config) *chap6Handler {
	return &chap6Handler{
		cfg: cfg,
	}
}
