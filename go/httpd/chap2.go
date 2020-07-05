package httpd

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

func (s *server) chap2(w http.ResponseWriter, r *http.Request) {
	uploadPath := path.Join("/tmp", "democry", "chap2")

	if err := os.MkdirAll(uploadPath, 0766); err != nil {
		log.Println("Error: ", err.Error())
		response(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	payload := struct {
		Content string `json:"content"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("Error: ", err.Error())
		response(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	// Data URL format:: https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/Data_URIs
	decoded, err := base64.StdEncoding.DecodeString(strings.Split(payload.Content, ",")[1])
	if err != nil {
		log.Println("Error: ", err.Error())
		response(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	// save file.
	tempFile, err := ioutil.TempFile(uploadPath, "upload-*.png")
	if err != nil {
		log.Println("Error: ", err.Error())
		response(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}
	defer tempFile.Close()

	// write this byte array to our temporary file
	tempFile.Write(decoded)

	response(w, http.StatusOK, map[string]string{
		"message": "success",
	})
}
