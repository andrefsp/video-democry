package httpd

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/andrefsp/video-democry/go/httpd/chap2"
	"github.com/andrefsp/video-democry/go/httpd/chap3"
	"github.com/andrefsp/video-democry/go/httpd/chap4"
	"github.com/andrefsp/video-democry/go/httpd/chap5"
)

func cors(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")

		h(w, r)
	}
}

type Config struct {
	StaticDir string
	SslMode   bool
	Hostname  string
	Port      string
}

type server struct {
	handler *mux.Router
	cfg     *Config
}

func (s *server) HttpHandler() http.Handler {
	// picture upload
	s.handler.HandleFunc("/chap2/endpoint", cors(chap2.New().Handler))

	// no op
	s.handler.HandleFunc("/chap3/endpoint", cors(chap3.New().Handler))

	// two user chat room
	s.handler.HandleFunc("/chap4/endpoint", cors(chap4.New().Handler))

	// multi user chat room
	s.handler.HandleFunc("/chap5/endpoint", cors(chap5.New().Handler))

	//
	s.handler.HandleFunc("/s/settings.js", s.SettingsHandler)

	// static files
	s.handler.PathPrefix("/s/").Handler(
		http.StripPrefix("/s/", http.FileServer(http.Dir(s.cfg.StaticDir))),
	)

	return s.handler
}

func NewServer(cfg *Config) *server {
	return &server{
		handler: mux.NewRouter(),
		cfg:     cfg,
	}
}
