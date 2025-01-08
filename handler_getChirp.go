package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (c *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {

	// Strip the "/api/chirps/" prefix to get just the chirpID
	path := strings.TrimPrefix(r.URL.Path, "/api/chirps/")

	// Ensure that what remains is a valid-looking UUID
	if path == "" {
		fmt.Println("UUID is empty")
		w.WriteHeader(404)
		return
	}

	chirpUUID, err := uuid.Parse(path)
	if err != nil {
		fmt.Println("Not a valid UUID")
		w.WriteHeader(404)
		return
	}

	chirp, err := c.dbQueries.GetChirp(r.Context(), chirpUUID)
	if err != nil {
		fmt.Println("Chirp not found")
		w.WriteHeader(404)
		return
	}

	response := ChirpJson{
		ID:        chirp.ID,
		UserId:    chirp.UserID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
	}

	writeJSONResponse(w, 200, response)
	return

}
