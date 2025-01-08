package config

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/brayanMuniz/Chirpy/internal/database"
)

type APIConfig struct {
	FileServerHits atomic.Int32
	DBQueries      *database.Queries
	Platform       string
	Secret         string
	PolkaKey       string
}

func (cfg *APIConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.FileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *APIConfig) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	msg := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.FileServerHits.Load())
	w.Write([]byte(msg))
}

func (cfg *APIConfig) HandleReset(w http.ResponseWriter, r *http.Request) {
	if cfg.Platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := cfg.DBQueries.DeleteAll(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	cfg.FileServerHits.Store(0)
	w.WriteHeader(http.StatusOK)
}

func (cfg *APIConfig) HandleGetChirps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"chirps": []}`))
}
