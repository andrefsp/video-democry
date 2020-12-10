package httpd

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/andrefsp/video-democry/go/config"

	"github.com/andrefsp/video-democry/go/httpd/chap2"
	"github.com/andrefsp/video-democry/go/httpd/chap3"
	"github.com/andrefsp/video-democry/go/httpd/chap4"
	"github.com/andrefsp/video-democry/go/httpd/chap5"
	"github.com/andrefsp/video-democry/go/httpd/chap6"
	"github.com/andrefsp/video-democry/go/httpd/chap7"
	"github.com/andrefsp/video-democry/go/httpd/chap8"
	"github.com/andrefsp/video-democry/go/httpd/chap9"
)

func cors(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")

		h(w, r)
	}
}

type server struct {
	handler *mux.Router
	cfg     *config.Config
}

func (s *server) HttpHandler() http.Handler {
	// picture upload
	chap2.New(s.cfg).RegisterHandlers(s.handler.PathPrefix("/chap2").Subrouter(), cors)

	// no op
	chap3.New(s.cfg).RegisterHandlers(s.handler.PathPrefix("/chap3").Subrouter(), cors)

	// two user chat room
	chap4.New(s.cfg).RegisterHandlers(s.handler.PathPrefix("/chap4").Subrouter(), cors)

	// multi user chat room
	chap5.New(s.cfg).RegisterHandlers(s.handler.PathPrefix("/chap5").Subrouter(), cors)

	// Video streaming
	chap6.New(s.cfg).RegisterHandlers(s.handler.PathPrefix("/chap6").Subrouter(), cors)

	// Multi user chat with relay server
	chap7.New(s.cfg).RegisterHandlers(s.handler.PathPrefix("/chap7").Subrouter(), cors)

	// Multiple video tracks on PeerConnection
	chap8.New(s.cfg).RegisterHandlers(s.handler.PathPrefix("/chap8").Subrouter(), cors)

	// Multi user chat with relay server, server initiator.
	chap9.New(s.cfg).RegisterHandlers(s.handler.PathPrefix("/chap9").Subrouter(), cors)

	// settings.js
	s.handler.HandleFunc("/s/settings.js", s.SettingsHandler)

	// static files
	s.handler.PathPrefix("/s/").Handler(
		http.StripPrefix("/s/", http.FileServer(http.Dir(s.cfg.StaticDir))),
	)

	return s.handler
}

func NewServer(cfg *config.Config) *server {
	return &server{
		handler: mux.NewRouter(),
		cfg:     cfg,
	}
}
