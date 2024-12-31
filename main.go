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

	"github.com/brayanMuniz/Chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // The underscore tells Go that you're importing it for its side effects, not because you need to use it.
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

	// /api/validate_chirp
	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body string `json:"body"` // NOTE: this must be capatilized in order to be exported, the value on the right is the name of what goes out
		}
		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			writeJSONResponse(w, 500, map[string]string{"error": "Something went wrong"})
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

		// message is good
		writeJSONResponse(w, 200, map[string]string{"cleaned_body": strings.Join(cleanResponseSlice, " ")})
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

		type CreateUserParams struct {
			ID    uuid.UUID
			Email string
		}

		// Create and populate the struct
		userParams := CreateUserParams{
			ID:    uuid.MustParse(uuid.NewString()), // Convert the string to UUID
			Email: params.Email,
		}

		// Call the function with the struct
		user, err := apiCfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams(userParams))

		if err != nil {
			fmt.Printf("Error creating user: %v\n", err)
			writeJSONResponse(w, 500, map[string]string{"error": "Failed to create user"})
			return
		}

		writeJSONResponse(w, 201, user)
		return

	})

	server.ListenAndServe()

}
