package main

import (
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
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		msg := (fmt.Sprintf("Hits: %d", apiCfg.fileserverHits.Load()))
		w.Write([]byte(msg))
	})

	// reset
	mux.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		apiCfg.fileserverHits.Store(0)
	})

	// healthz
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		msg := []byte("OK")
		w.Write(msg)
	})

	server.ListenAndServe()

}
