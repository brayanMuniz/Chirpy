package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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
	mux := http.NewServeMux() // responsible for handling and routing paths
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	} // listens to network address and handles it with mux

	apiCfg := apiConfig{} // this inits it to its 0 value

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

	// reset
	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		apiCfg.fileserverHits.Store(0)
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

		// message is good
		writeJSONResponse(w, 200, map[string]bool{"valid": true})
		return

	})

	server.ListenAndServe()

}
