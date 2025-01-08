package main

import (
	"encoding/json"
	"net/http"

	"github.com/brayanMuniz/Chirpy/internal/auth"
	"github.com/google/uuid"
)

func (c *apiConfig) handlerWebHooks(w http.ResponseWriter, r *http.Request) {
	// check authorization using the provided api key
	apiToken, err := auth.GetAPIKey(r.Header)
	if err != nil {
		writeJSONResponse(w, 401, map[string]string{"error": "Unathorized"})
		return
	}

	if apiToken != c.polkakey {
		writeJSONResponse(w, 401, map[string]string{"error": "Unathorized"})
		return
	}

	type Data struct {
		UserId string `json:"user_id"`
	}

	type parameters struct {
		Event string `json:"event"`
		Data  Data   `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		writeJSONResponse(w, 500, map[string]string{"error": "Something went wrong"})
		return
	}

	if params.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	parsedID, err := uuid.Parse(params.Data.UserId)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	err = c.dbQueries.UpgradeToChirpyRed(r.Context(), parsedID)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	w.WriteHeader(204)

}
