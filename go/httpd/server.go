package httpd

import (
	"encoding/json"
	"net/http"
)

func response(w http.ResponseWriter, responseCode int, payload interface{}) {
	jData, err := json.Marshal(payload)
	if err != nil {
		return
	}

	w.WriteHeader(responseCode)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jData)

}

func cors(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")

		h(w, r)
	}
}

type server struct {
	handler *http.ServeMux
}

func (s *server) HttpHandler() http.Handler {
	s.handler.HandleFunc("/chap2/endpoint", cors(s.chap2))
	return s.handler
}

func NewServer() *server {
	return &server{
		handler: http.NewServeMux(),
	}
}
