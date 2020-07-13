package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"

	"github.com/andrefsp/video-democry/go/httpd"
	"github.com/andrefsp/video-democry/go/stunturn"
)

func getPWD() string {
	if os.Getenv("V_PATH") != "" {
		return os.Getenv("V_PATH")
	}

	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("no runtime ok")
	}

	return path.Dir(filename)
}

func relPath(parts ...string) string {
	pwd := getPWD()

	parts = append([]string{pwd}, parts...)
	return path.Join(parts...)
}

func valueOrDefault(val, default_ string) string {
	if val == "" {
		return default_
	}
	return val
}

var sslMode = valueOrDefault(os.Getenv("SSL"), "false") == "true"

var listenAddr = valueOrDefault(os.Getenv("LISTEN_ADDR"), "0.0.0.0")

var listenPort = valueOrDefault(os.Getenv("LISTEN_PORT"), "8081")

var sslDir = "ssl/"

var staticDir = relPath("../fe/src/")

var hostname = valueOrDefault(os.Getenv("V_HOSTNAME"), "localhost")

var relayAddr = valueOrDefault(os.Getenv("RELAY_ADDR"), "192.168.0.39")

func main() {
	go stunturn.Start(hostname, relayAddr)

	s := httpd.NewServer(&httpd.Config{
		StaticDir: staticDir,
		SslMode:   sslMode,
		Hostname:  hostname,
		Port:      listenPort,
	})

	fullListenAddr := fmt.Sprintf("%s:%s", listenAddr, listenPort)

	log.Printf("serving on '%s' sslMode: %b", fullListenAddr, sslMode)
	switch sslMode {
	case true:
		log.Println("Serving over https")
		log.Fatal(http.ListenAndServeTLS(
			fullListenAddr,
			relPath(sslDir, "private.crt"),
			relPath(sslDir, "private.key"),
			s.HttpHandler(),
		))
	default:
		log.Println("Serving over http")
		log.Fatal(http.ListenAndServe(fullListenAddr, s.HttpHandler()))
	}

}
