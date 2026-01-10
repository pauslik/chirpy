package main

import (
	"fmt"
	"net/http"
)

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, req *http.Request) {
	body := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
	respondWithHTML(w, 200, body)
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, req *http.Request) {
	// Reset page visit stat
	cfg.fileserverHits.Store(0)

	// Reset user database
	cfg.db.ResetUsers(req.Context())

	// Reset chirp database
	cfg.db.ResetChirps(req.Context())

	body := "OK\n"
	respondWithText(w, 200, body)
}
