package main

import (
	"fmt"
	"net/http"
)

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(403)
		return
	}

	err := cfg.dbQueries.DeleteAll(r.Context())
	if err != nil {
		fmt.Println("Failed to delete everything")
		w.WriteHeader(500)
		return
	}

	fmt.Println("DELETING EVERYTHING")
	w.WriteHeader(200)
	cfg.fileserverHits.Store(0)
	return
}
