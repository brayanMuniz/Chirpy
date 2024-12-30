package main

import "net/http"

func main() {
	mux := http.NewServeMux() // responsible for handling and routing paths
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	} // listens to network address and handles it with mux

	// Serve static files from the current directory under the /app/ path
	// StripPrefix removes /app from the request path before looking for files
	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("./"))))

	// healthz
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		msg := []byte("OK")
		w.Write(msg)
	})

	server.ListenAndServe()

}
