package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	p, err := bcrypt.GenerateFromPassword([]byte(password), 4) // NOTE: Minimum is 4
	if err != nil {
		return "", err
	}

	return string(p), nil
}

func CheckPasswordHash(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return err
	}
	return nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// NOTE: Why []byte?: https://golang-jwt.github.io/jwt/usage/signing_methods/#signing-methods-and-key-types
	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{} // the claims will be filled out from the callback function
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil // returns the "decrypted" key
	})

	if err != nil {
		fmt.Println("Parsing Error:", err)
		return uuid.Nil, err
	}

	// Not a valid token
	if claims.Issuer != "chirpy" || !time.Now().Before(claims.ExpiresAt.Time) || !token.Valid {
		return uuid.Nil, errors.New("401 Unauthorized")
	}

	userUUID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}
	return userUUID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	bearerToken := headers.Get("Authorization")
	if bearerToken == "" || bearerToken[:7] != "Bearer " {
		return "", errors.New("Bearer Token not found")
	}
	token := bearerToken[7:]
	return token, nil
}
