package main

import (
	"log"
	"net/http"

	"github.com/andrefsp/video-democry/go/httpd"
)

func main() {

	s := httpd.NewServer()

	log.Println("serving on 8081...")
	log.Fatal(http.ListenAndServe(":8081", s.HttpHandler()))
}