package responses

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Message string `json:"message"`
}

func NewError(message string) *Error {
	return &Error{
		Message: message,
	}
}

func Send(w http.ResponseWriter, responseCode int, payload interface{}) {
	jData, err := json.Marshal(payload)
	if err != nil {
		return
	}

	w.WriteHeader(responseCode)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jData)
}
