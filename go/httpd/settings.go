package httpd

import (
	"fmt"
	"net/http"
)

func (s *server) SettingsHandler(w http.ResponseWriter, r *http.Request) {

	wsProtocol := "ws"
	if s.cfg.SslMode {
		wsProtocol = "wss"
	}

	fullHostname := fmt.Sprintf("%s:%s", s.cfg.Hostname, s.cfg.Port)

	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte(fmt.Sprintf(`
		export const wsURL = "%s://%s";
	`, wsProtocol, fullHostname)))
}
