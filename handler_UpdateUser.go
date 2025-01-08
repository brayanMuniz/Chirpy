package main

import (
	"encoding/json"
	"github.com/brayanMuniz/Chirpy/internal/auth"
	"github.com/brayanMuniz/Chirpy/internal/database"
	"github.com/google/uuid"
	"net/http"
	"time"
)

func (c *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
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

	// check provided parameters
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		writeJSONResponse(w, 500, map[string]string{"error": "Could not decode your request"})
		return
	}

	// hash the password
	hPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		writeJSONResponse(w, 500, map[string]string{"error": "Failed to hash your password"})
		return
	}

	// update the email and password
	user, err := c.dbQueries.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:          params.Email,
		HashedPassword: hPassword,
		ID:             userID,
	})
	if err != nil {
		writeJSONResponse(w, 500, map[string]string{"error": "Failed to save your new information to the database"})
		return
	}

	// Send back the data
	type userResponse struct {
		ID          uuid.UUID `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}
	response := userResponse{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}

	writeJSONResponse(w, 200, response)
}
