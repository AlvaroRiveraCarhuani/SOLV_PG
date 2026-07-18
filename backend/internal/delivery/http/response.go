package httpdelivery

import (
	"encoding/json"
	"net/http"
)

type GlobalResponse struct {
	Data    any    `json:"data"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func SendJSON(w http.ResponseWriter, status int, data any, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(GlobalResponse{
		Data:    data,
		Error:   "",
		Message: msg,
	})
}

func SendError(w http.ResponseWriter, status int, err string, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(GlobalResponse{
		Data:    nil,
		Error:   err,
		Message: msg,
	})
}
