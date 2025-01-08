package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/brayanMuniz/Chirpy/internal/auth"
	"github.com/google/uuid"
)

func (c *apiConfig) deleteChirp(w http.ResponseWriter, r *http.Request) {
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

	// Get chirp from database
	chirp, err := c.dbQueries.GetChirp(r.Context(), chirpUUID)
	if err != nil {
		fmt.Println("Chirp not found")
		w.WriteHeader(404)
		return
	}

	// authenticate the user using their JWT
	userToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		writeJSONResponse(w, 401, map[string]string{"error": "Unathorized"})
		return
	}

	userID, err := auth.ValidateJWT(userToken, c.secret)
	if err != nil {
		writeJSONResponse(w, 401, map[string]string{"error": "Invalid user_id format"})
		return
	}

	// check that the chirp author is the requesting author
	if chirp.UserID != userID {
		writeJSONResponse(w, 403, map[string]string{"error": "You are not the chirp author"})
		return
	}

	err = c.dbQueries.DeleteChirp(r.Context(), chirp.ID)
	if err != nil {
		writeJSONResponse(w, 404, map[string]string{"error": "Could not find the chirp"})
		return
	}
	w.WriteHeader(204)

}
