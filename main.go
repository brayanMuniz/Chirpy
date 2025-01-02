package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/brayanMuniz/Chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // The underscore tells Go that you're importing it for its side effects, not because you need to use it.
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}

// Database structs
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	User_id   uuid.UUID `json:"user_id"`
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

	apiCfg := apiConfig{} // this inits it to its 0 value
	apiCfg.dbQueries = dbQuries
	apiCfg.platform = platform

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

	// TODO: The main problem is how the test is being generated. It is using a template in the user_id

	// /api/chirps
	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body   string `json:"body"` // NOTE: this must be capatilized in order to be exported, the value on the right is the name of what goes out
			UserID string `json:"user_id"`
		}
		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			writeJSONResponse(w, 500, map[string]string{"error": "Could not decode your request"})
			return
		}

		fmt.Printf("Received params: %+v\n", params)

		// Convert string to UUID
		userID, err := uuid.Parse(params.UserID)
		if err != nil {
			writeJSONResponse(w, 400, map[string]string{"error": "Invalid user_id format"})
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
			Email string `json:"email"`
		}
		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			writeJSONResponse(w, 500, map[string]string{"error": "Something went wrong"})
			return
		}

		userParams := database.CreateUserParams{
			ID:    uuid.New(), // This generates a new UUID
			Email: params.Email,
		}

		// Call the function with the struct
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

	server.ListenAndServe()

}
