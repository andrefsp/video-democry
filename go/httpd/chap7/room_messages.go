package chap7

import "github.com/pion/webrtc/v3"

// messages
type InfoMessage struct {
	Uri     string `json:"uri"`
	Message string `json:"message"`
}

type message struct {
	Uri string `json:"uri"`
}

type OutNegotiationNeeded struct {
	Uri    string `json:"uri"`
	ToUser *user  `json:"toUser"`
}

type InICECandidate struct {
	FromUser  *user                   `json:"fromUser"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type OutICECandidate struct {
	Uri       string                  `json:"uri"`
	ToUser    *user                   `json:"toUser"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type InOffer struct {
	FromUser *user                     `json:"fromUser"`
	Offer    webrtc.SessionDescription `json:"offer"`
}

type OutOffer struct {
	Uri    string                    `json:"uri"`
	ToUser *user                     `json:"fromUser"`
	Offer  webrtc.SessionDescription `json:"offer"`
}

type InAnswer struct {
	FromUser *user                     `json:"fromUser"`
	Answer   webrtc.SessionDescription `json:"answer"`
}

type OutAnswer struct {
	Uri    string                    `json:"uri"`
	ToUser *user                     `json:"toUser"`
	Answer webrtc.SessionDescription `json:"answer"`
}

type InUserJoinMessage struct {
	User *user `json:"user"`
}

type OutUserEventMessage struct {
	Uri   string  `json:"uri"`
	User  *user   `json:"user"`
	Users []*user `json:"roomUsers"`
}
