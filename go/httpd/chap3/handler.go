package chap3

import (
	"log"
	"net/http"
	"os"
	"path"

	"github.com/andrefsp/video-democry/go/config"
	"github.com/andrefsp/video-democry/go/httpd/responses"
	"github.com/gorilla/mux"
)

type chap3Handler struct {
	cfg *config.Config
}

func (s *chap3Handler) Handler(w http.ResponseWriter, r *http.Request) {
	uploadPath := path.Join("/tmp", "democry", "chap3")

	if err := os.MkdirAll(uploadPath, 0766); err != nil {
		log.Println("Error: ", err.Error())
		responses.Send(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	responses.Send(w, http.StatusOK, map[string]string{
		"message": "success",
	})
}

func (s *chap3Handler) RegisterHandlers(m *mux.Router, middleware func(h http.HandlerFunc) http.HandlerFunc) {
	m.HandleFunc("/endpoint", s.Handler)
}

func New(cfg *config.Config) *chap3Handler {
	return &chap3Handler{
		cfg: cfg,
	}
}
