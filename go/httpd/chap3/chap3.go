package chap3

import (
	"log"
	"net/http"
	"os"
	"path"

	"github.com/andrefsp/video-democry/go/httpd/responses"
)

type chap3Handler struct{}

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

func New() *chap3Handler {
	return &chap3Handler{}
}
