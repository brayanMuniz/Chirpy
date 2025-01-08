package main

import (
	"fmt"
	"github.com/brayanMuniz/Chirpy/internal/auth"
	"github.com/brayanMuniz/Chirpy/internal/database"
	"net/http"
	"time"
)

func (c *apiConfig) handlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		writeJSONResponse(w, 401, map[string]string{"error": "Provide a refresh_token"})
		return
	}

	t, err := c.dbQueries.GetRefreshToken(r.Context(), refreshToken)
	if err != nil {
		writeJSONResponse(w, 401, map[string]string{"error": "Could not find your refresh_token"})
		return
	}

	if !time.Now().Before(t.ExpiresAt) || t.RevokedAt.Valid {
		writeJSONResponse(w, 401, map[string]string{"error": "Token has expired"})
		return
	}

	// create a new JWT and return that
	tokenString, err := auth.MakeJWT(t.UserID, c.secret, time.Duration(3600)*time.Second) // NOTE: needs to be multiplied this way in order for it to work
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
		UserID: t.UserID,
	})
	if err != nil {
		writeJSONResponse(w, 500, map[string]string{"error": "Error in the db, my bad"})
		return
	}

	writeJSONResponse(w, 200, map[string]string{"token": tokenString})
	return
}
