package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/brayanMuniz/Chirpy/internal/auth"
	"github.com/brayanMuniz/Chirpy/internal/database"
)

func (c *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
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

	// check that the user exist and that the password is correct
	user, err := c.dbQueries.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		fmt.Println("Could not find user")
		writeJSONResponse(w, 401, map[string]string{"error": "Incorrect email or password"})
		return

	}

	err = auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		fmt.Println("Password does not match")
		writeJSONResponse(w, 401, map[string]string{"error": "Incorrect email or password"})
		return
	}

	// generate and respond with the token
	tokenString, err := auth.MakeJWT(user.ID, c.secret, time.Duration(3600)*time.Second) // NOTE: needs to be multiplied this way in order for it to work
	if err != nil {
		fmt.Println("Could not generate token for user")
		writeJSONResponse(w, 500, map[string]string{"error": "Could not generate token"})
		return
	}

	// Make refresh token and add it to the database
	rToken, err := auth.MakeRefreshToken()
	if err != nil {
		writeJSONResponse(w, 500, map[string]string{"error": "Could not generate refresh token"})
		return
	}
	_, err = c.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:  rToken,
		UserID: user.ID,
	})
	if err != nil {
		writeJSONResponse(w, 500, map[string]string{"error": "Error in the db, my bad"})
		return
	}

	userResponse := UserJson{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        tokenString,
		RefreshToken: rToken,
		IsChirpyRed:  user.IsChirpyRed,
	}

	writeJSONResponse(w, 200, userResponse)
	return

}
