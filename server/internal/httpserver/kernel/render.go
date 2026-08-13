package kernel

import (
	"encoding/json"
	"net/http"
)

const jsonContentType = "application/json; charset=utf-8"

// EncodeJSON writes v as JSON with the same Content-Type the hand-rolled
// handlers set. A nil v with status 204 writes no body.
func EncodeJSON(w http.ResponseWriter, status int, v any) {
	if status == 0 {
		status = http.StatusOK
	}
	if status == http.StatusNoContent || v == nil {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", jsonContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// EncodeOK is EncodeJSON with 200.
func EncodeOK(w http.ResponseWriter, v any) {
	EncodeJSON(w, http.StatusOK, v)
}

// EncodeCreated is EncodeJSON with 201.
func EncodeCreated(w http.ResponseWriter, v any) {
	EncodeJSON(w, http.StatusCreated, v)
}
