package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/brayanMuniz/Chirpy/internal/auth"
	"github.com/brayanMuniz/Chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // The underscore tells Go that you're importing it for its side effects, not because you need to use it.
	"net/http"
	"os"
	"os/exec"
	"sync/atomic"
	"time"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
	secret         string
	polkakey       string
}

// Database structs
type UserJson struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
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

func runMigrations(dbURL string) error {
	cmd := exec.Command("goose", "-dir", "./sql/schema", "postgres", dbURL, "up")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	godotenv.Load()

	// Load in the database
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		fmt.Println("DB_URL not found")
		return
	}

	// Run migrations
	if err := runMigrations(dbURL); err != nil {
		fmt.Println("Failed to run migrations:", err)
		return
	}

	db, err := sql.Open("postgres", dbURL)
	if err = db.Ping(); err != nil {
		fmt.Println("Error connecting to the database: ", err)
		return
	}

	mux := http.NewServeMux() // responsible for handling and routing paths
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	} // listens to network address and handles it with mux

	// apiCfg
	apiCfg := apiConfig{}
	apiCfg.dbQueries = database.New(db)
	apiCfg.platform = os.Getenv("PLATFORM")
	apiCfg.secret = os.Getenv("SECRET")
	apiCfg.polkakey = os.Getenv("POLKA_KEY")

	// Serve static files from the /app/static directory under the /app/ path
	fileServer := http.FileServer(http.Dir("./static")) // NOTE: if you are running this without docker, change this to ./
	handler := http.StripPrefix("/app", fileServer)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	// server status
	mux.HandleFunc("GET /api/healthz", apiCfg.handlerHealthz)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)

	// dont allow this to happen unless the env variable is set to dev
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)

	// GET /api/chirps
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetAllChirps)

	// GET /api/chirps/{chirpID}
	mux.HandleFunc("GET /api/chirps/", apiCfg.handlerGetChirp)

	// DELETE /api/chirps/{chirpID}
	mux.HandleFunc("DELETE /api/chirps/", apiCfg.deleteChirp)

	// POst /api/chirps
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerPostChirp)

	// PUT /api/users
	mux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateUser)

	// POST /api/users
	mux.HandleFunc("POST /api/users", apiCfg.handlerPostUser)

	// POST /api/refresh
	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefreshToken)

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
	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)

	// POST /api/polka/webhooks
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerWebHooks)

	server.ListenAndServe()

}
