package handlers

import "net/http"

type API interface {
	MiddlewareMetricsInc(http.Handler) http.Handler
	HandleMetrics(http.ResponseWriter, *http.Request)
	HandleReset(http.ResponseWriter, *http.Request)
	HandleGetChirps(http.ResponseWriter, *http.Request)
}

func SetupRoutes(api API) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/app/", api.MiddlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./")))))
	mux.HandleFunc("/admin/metrics", api.HandleMetrics)
	mux.HandleFunc("/admin/reset", api.HandleReset)
	mux.HandleFunc("/api/chirps", api.HandleGetChirps)

	return mux
}

