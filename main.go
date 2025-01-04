package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/brayanMuniz/Chirpy/internal/auth"
	"github.com/brayanMuniz/Chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // The underscore tells Go that you're importing it for its side effects, not because you need to use it.
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
	secret         string
}

// Database structs
type UserJson struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

type ChirpJson struct {
	ID        uuid.UUID `json:"id"`
	UserId    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
}

// this is the method in order to increment the apiConfig by one
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	// Return a NEW handler
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is where we can do things BEFORE handling the request
		cfg.fileserverHits.Add(1)

		// This is how we call the next handler
		next.ServeHTTP(w, r)

		// This is where we can do things AFTER handling the request

	})
}

// to keep code DRY
func writeJSONResponse(w http.ResponseWriter, statusCode int, respBody interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	dat, err := json.Marshal(respBody)
	if err != nil {
		w.WriteHeader(500) // Internal server error
		w.Write([]byte(`{"error": "Internal server error"}`))
		return
	}
	w.Write(dat)
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)
	if err = db.Ping(); err != nil {
		fmt.Println("Error connecting to the database: ", err)
		return
	}
	dbQuries := database.New(db) // returns a pointer to it

	mux := http.NewServeMux() // responsible for handling and routing paths
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	} // listens to network address and handles it with mux

	// apiCfg
	apiCfg := apiConfig{} // this inits it to its 0 value
	apiCfg.dbQueries = dbQuries
	apiCfg.platform = platform
	apiCfg.secret = os.Getenv("SECRET")

	// Serve static files from the current directory under the /app/ path
	// StripPrefix removes /app from the request path before looking for files
	fileServer := http.FileServer(http.Dir("./"))
	handler := http.StripPrefix("/app", fileServer)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	// metrics
	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		msg := (fmt.Sprintf("<html> <body> <h1>Welcome, Chirpy Admin</h1> <p>Chirpy has been visited %d times!</p> </body> </html>", apiCfg.fileserverHits.Load()))
		w.Write([]byte(msg))
	})

	// POST /admin/reset
	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		// dont allow this to happen unless the env variable is set to dev
		if apiCfg.platform != "dev" {
			w.WriteHeader(403)
			return
		}

		err := apiCfg.dbQueries.DeleteAll(r.Context())
		if err != nil {
			w.WriteHeader(500)
			return
		}
		fmt.Println("DELETING EVERYTHING")
		w.WriteHeader(200)
		apiCfg.fileserverHits.Store(0)
		return
	})

	// healthz
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		msg := []byte("OK")
		w.Write(msg)
	})

	// GET /api/chirps
	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		chirps, err := apiCfg.dbQueries.GetAllChirps(r.Context())
		if err != nil {
			w.WriteHeader(500)
			return
		}
		chirpArray := []ChirpJson{}

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
	})

	// GET /api/chirps/{chirpID}
	mux.HandleFunc("/api/chirps/", func(w http.ResponseWriter, r *http.Request) {
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

		chirp, err := apiCfg.dbQueries.GetChirp(r.Context(), chirpUUID)
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
	})

	// /api/chirps
	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
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

		userID, err := auth.ValidateJWT(userToken, apiCfg.secret)
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
		chirp, err := apiCfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams(chirpParams))
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

	})

	// POST /api/users
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
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
		user, err := apiCfg.dbQueries.CreateUser(r.Context(), userParams)
		if err != nil {
			fmt.Printf("Error creating user: %v\n", err)
			writeJSONResponse(w, 500, map[string]string{"error": "Failed to create user"})
			return
		}

		// wrapper
		type userResponse struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Email     string    `json:"email"`
		}
		response := userResponse{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		}

		writeJSONResponse(w, 201, response)

		return

	})

	// POST /api/refresh
	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		refreshToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			writeJSONResponse(w, 401, map[string]string{"error": "Provide a refresh_token"})
			return
		}

		t, err := apiCfg.dbQueries.GetRefreshToken(r.Context(), refreshToken)
		if err != nil {
			writeJSONResponse(w, 401, map[string]string{"error": "Could not find your refresh_token"})
			return
		}

		if !time.Now().Before(t.ExpiresAt) || t.RevokedAt.Valid {
			writeJSONResponse(w, 401, map[string]string{"error": "Token has expired"})
			return
		}

		// create a new JWT and return that
		tokenString, err := auth.MakeJWT(t.UserID, apiCfg.secret, time.Duration(3600)*time.Second) // NOTE: needs to be multiplied this way in order for it to work
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
		_, err = apiCfg.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
			Token:  rToken,
			UserID: t.UserID,
		})
		if err != nil {
			writeJSONResponse(w, 500, map[string]string{"error": "Error in the db, my bad"})
			return
		}

		writeJSONResponse(w, 200, map[string]string{"token": tokenString})
		return
	})

	// POST /api/revoke
	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		refreshToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			writeJSONResponse(w, 401, map[string]string{"error": "Provide a refresh_token"})
			return
		}
		_, err = apiCfg.dbQueries.RevokeToken(r.Context(), refreshToken)
		if err != nil {
			writeJSONResponse(w, 500, map[string]string{"error": "Could not revoke the access token"})
			return
		}

		w.WriteHeader(204)
	})

	// POST /api/login
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
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
		user, err := apiCfg.dbQueries.GetUserByEmail(r.Context(), params.Email)
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
		tokenString, err := auth.MakeJWT(user.ID, apiCfg.secret, time.Duration(3600)*time.Second) // NOTE: needs to be multiplied this way in order for it to work
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
		_, err = apiCfg.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
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
		}

		writeJSONResponse(w, 200, userResponse)
		return

	})

	server.ListenAndServe()

}
