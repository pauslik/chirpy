package main

import (
	"database/sql"

	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pauslik/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	jwt            string
}

func main() {
	apiCfg := apiConfig{}

	// Load environment variables
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	jwtSecret := os.Getenv("JWT_SECRET")

	// Load the database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Could not open a connection to database. %v", err)
		os.Exit(1)
	}
	dbQueries := database.New(db)

	// Save to config
	apiCfg.db = dbQueries
	apiCfg.jwt = jwtSecret

	// HTTP request multiplexer
	mux := http.NewServeMux()

	// APP handlers
	// mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	appHandler := http.FileServer(http.Dir("."))
	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(appHandler)))
	assetsHandler := http.FileServer(http.Dir("./app/assets"))
	mux.Handle("/app/assets", apiCfg.middlewareMetricsInc(assetsHandler))

	//ADMIN handlers
	// Display metrics endpoint
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	// Reset metrics endpoint
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)

	// API handlers
	// Server health check endpoint
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		body := "OK"
		w.Write([]byte(body))
	})
	// Users
	mux.HandleFunc("POST /api/users", apiCfg.usersHandler)
	mux.HandleFunc("POST /api/login", apiCfg.loginHandler)
	mux.HandleFunc("POST /api/refresh", apiCfg.refreshHandler)
	mux.HandleFunc("POST /api/revoke", apiCfg.revokeHandler)
	// Chirps
	mux.HandleFunc("POST /api/chirps", apiCfg.createChirpHandler)
	mux.HandleFunc("GET /api/chirps", apiCfg.getChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpIDHandler)

	// HTTP server
	server := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	// Start the server
	server.ListenAndServe()

}
