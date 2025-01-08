package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/brayanMuniz/Chirpy/internal/auth"
	"github.com/brayanMuniz/Chirpy/internal/database"
	"github.com/google/uuid"
)

func (c *apiConfig) handlerPostUser(w http.ResponseWriter, r *http.Request) {

	// decoding the request
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		writeJSONResponse(w, 500, map[string]string{"error": "Something went wrong"})
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		writeJSONResponse(w, 500, map[string]string{"error": "Something went wrong"})
		return
	}

	userParams := database.CreateUserParams{
		ID:             uuid.New(),
		Email:          params.Email,
		HashedPassword: hashedPassword,
	}

	// create user in db
	user, err := c.dbQueries.CreateUser(r.Context(), userParams)
	if err != nil {
		fmt.Printf("Error creating user: %v\n", err)
		writeJSONResponse(w, 500, map[string]string{"error": "Failed to create user"})
		return
	}

	// wrapper
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
		IsChirpyRed: false,
	}

	writeJSONResponse(w, 201, response)

	return

}
