package main

import (
	"net/http"
)

func (cfg *apiConfig) handlerHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	msg := []byte("OK")
	w.Write(msg)
}
