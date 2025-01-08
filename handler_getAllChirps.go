package main

import (
	"fmt"
	"net/http"

	"github.com/brayanMuniz/Chirpy/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {

	query := database.GetChirpsParams{}
	chirpArray := []ChirpJson{}

	// Handle author_id query
	author_id := r.URL.Query().Get("author_id")
	var queryUUID uuid.UUID
	if author_id != "" {
		parsed_id, err := uuid.Parse(author_id)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		queryUUID = parsed_id
	}
	query.Column1 = queryUUID

	// Handle sort query
	sortBy := r.URL.Query().Get("sort")
	query.Column2 = "asc" // default
	if sortBy == "desc" || sortBy == "asc" {
		query.Column2 = sortBy
	}

	fmt.Print(query.Column1)

	chirps, err := cfg.dbQueries.GetChirps(r.Context(), query)

	if err != nil {
		w.WriteHeader(500)
		return
	}

	for _, chirp := range chirps {
		response := ChirpJson{
			ID:        chirp.ID,
			UserId:    chirp.UserID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
		}
		chirpArray = append(chirpArray, response)

	}

	writeJSONResponse(w, 200, chirpArray)
	return

}
