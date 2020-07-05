package httpd

import (
	"log"
	"net/http"
	"os"
	"path"
)

func (s *server) chap3(w http.ResponseWriter, r *http.Request) {
	uploadPath := path.Join("/tmp", "democry", "chap3")

	if err := os.MkdirAll(uploadPath, 0766); err != nil {
		log.Println("Error: ", err.Error())
		response(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	response(w, http.StatusOK, map[string]string{
		"message": "success",
	})
}
