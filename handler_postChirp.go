package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/brayanMuniz/Chirpy/internal/auth"
	"github.com/brayanMuniz/Chirpy/internal/database"
	"github.com/google/uuid"
)

func (c *apiConfig) handlerPostChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		writeJSONResponse(w, 500, map[string]string{"error": "Could not decode your request"})
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

	// msg too long
	if len(params.Body) > 140 {
		writeJSONResponse(w, 400, map[string]string{"error": "Chirp is too long"})
		return
	}

	// check for bad words
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	cleanResponseSlice := []string{}
	for _, v := range strings.Split(params.Body, " ") {
		if slices.Contains(badWords, strings.ToLower(v)) {
			cleanResponseSlice = append(cleanResponseSlice, "****")
		} else {
			cleanResponseSlice = append(cleanResponseSlice, v)
		}
	}

	// NOTE: Use the params from SQLC
	chirpParams := database.CreateChirpParams{
		ID:     uuid.New(), // This generates a new UUID
		UserID: userID,
		Body:   strings.Join(cleanResponseSlice, " "), // Note: field name is Body, not body
	}

	// Call the function with the struct
	chirp, err := c.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams(chirpParams))
	if err != nil {
		fmt.Printf("Error creating chirp: %v\n", err)
		writeJSONResponse(w, 500, map[string]string{"error": "Failed to create chirp"})
		return
	}

	// wrapper
	type chirpResponse struct {
		ID        uuid.UUID `json:"id"`
		UserId    uuid.UUID `json:"user_id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
	}
	response := chirpResponse{
		ID:        chirp.ID,
		UserId:    chirp.UserID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
	}

	writeJSONResponse(w, 201, response)
	return

}
