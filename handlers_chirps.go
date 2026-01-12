package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pauslik/chirpy/internal/auth"
	"github.com/pauslik/chirpy/internal/database"
)

type inputChirp struct {
	Body string `json:"body"`
	// UserID uuid.UUID `json:"user_id"`
}

type outputChirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (c *inputChirp) cleanBody() {
	splitBody := strings.Split(c.Body, " ")
	replacements := map[string]string{
		"kerfuffle": "****",
		"sharbert":  "****",
		"fornax":    "****",
	}
	for i, word := range splitBody {
		for target, replacement := range replacements {
			if strings.EqualFold(word, target) {
				splitBody[i] = replacement
			}
		}
	}
	cleanBody := strings.Join(splitBody, " ")
	c.Body = cleanBody
}

func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, req *http.Request) {
	iChirp := inputChirp{}
	oChirp := outputChirp{}

	// Decoding input
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&iChirp)
	if err != nil {
		fErr := fmt.Sprintf("Error decoding input: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	// Auth
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fErr := fmt.Sprintf("Error getting bearer: %s", err)
		respondWithText(w, 500, fErr)
		return
	}
	authID, err := auth.ValidateJWT(token, cfg.jwt)
	if err != nil {
		fErr := fmt.Sprintf("Error validating %s JWT: %s", authID.String(), err)
		respondWithText(w, 401, fErr)
		return
	}

	// Process Chirp
	if len(iChirp.Body) > 140 {
		respondWithText(w, 400, "Chirp is too long")
		return
	}
	iChirp.cleanBody()

	// Create Chirp
	ccp := database.CreateChirpParams{
		Body:   iChirp.Body,
		UserID: authID,
	}
	dbChirp, err := cfg.db.CreateChirp(req.Context(), ccp)
	if err != nil {
		fErr := fmt.Sprintf("Error creating chirp: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	oChirp.Body = dbChirp.Body
	oChirp.UserID = dbChirp.UserID
	oChirp.ID = dbChirp.ID
	oChirp.CreatedAt = dbChirp.CreatedAt
	oChirp.UpdatedAt = dbChirp.UpdatedAt

	respondWithJSON(w, 201, oChirp)
}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, req *http.Request) {
	chirps := []outputChirp{}

	aID := req.URL.Query().Get("author_id")

	var dbChirps []database.Chirp
	if aID == "" {
		var err error
		dbChirps, err = cfg.db.GetChirps(req.Context())
		if err != nil {
			fErr := fmt.Sprintf("Error getting chirps: %s", err)
			respondWithText(w, 500, fErr)
			return
		}
	} else {
		userID, err := uuid.Parse(aID)
		if err != nil {
			fErr := fmt.Sprintf("Error parsing author_id: %s", err)
			respondWithText(w, 500, fErr)
			return
		}
		dbChirps, err = cfg.db.GetChirpsUser(req.Context(), userID)
		if err != nil {
			fErr := fmt.Sprintf("Error getting chirps: %s", err)
			respondWithText(w, 500, fErr)
			return
		}
	}

	// This feels like a terrible aproach to this problem, maybe use the SQLC option?
	for _, dbc := range dbChirps {
		chirp := outputChirp{}
		chirp.Body = dbc.Body
		chirp.UserID = dbc.UserID
		chirp.ID = dbc.ID
		chirp.CreatedAt = dbc.CreatedAt
		chirp.UpdatedAt = dbc.UpdatedAt
		chirps = append(chirps, chirp)
	}

	// Sort desc if ?sort=desc was provided in the request
	sortOrder := req.URL.Query().Get("sort")
	if sortOrder == "desc" {
		sort.Slice(chirps, func(i, j int) bool { return chirps[i].CreatedAt.Compare(chirps[j].CreatedAt) > 0 })
	}

	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) getChirpIDHandler(w http.ResponseWriter, req *http.Request) {
	chirp := outputChirp{}

	chirpID, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		fErr := fmt.Sprintf("Not valid Chirp ID: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	dbChirp, err := cfg.db.GetChirp(req.Context(), chirpID)
	if err != nil {
		// Handle not found error
		if errors.Is(err, sql.ErrNoRows) {
			respondWithText(w, 404, "Chirp not found")
			return
		}
		fErr := fmt.Sprintf("Error getting chirps: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	chirp.Body = dbChirp.Body
	chirp.UserID = dbChirp.UserID
	chirp.ID = dbChirp.ID
	chirp.CreatedAt = dbChirp.CreatedAt
	chirp.UpdatedAt = dbChirp.UpdatedAt

	respondWithJSON(w, 200, chirp)
}

func (cfg *apiConfig) deleteChirpIDHandler(w http.ResponseWriter, req *http.Request) {
	// Auth
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fErr := fmt.Sprintf("Error getting bearer: %s", err)
		respondWithText(w, 401, fErr)
		return
	}
	authID, err := auth.ValidateJWT(token, cfg.jwt)
	if err != nil {
		fErr := fmt.Sprintf("Error validating %s JWT: %s", authID.String(), err)
		respondWithText(w, 401, fErr)
		return
	}

	// Check and parse Chirp ID
	chirpID, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		fErr := fmt.Sprintf("Not valid Chirp ID: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	// Get chirp from database
	dbChirp, err := cfg.db.GetChirp(req.Context(), chirpID)
	if err != nil {
		// Handle not found error
		if errors.Is(err, sql.ErrNoRows) {
			respondWithText(w, 404, "Chirp not found")
			return
		}
		fErr := fmt.Sprintf("Error getting chirps: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	if authID != dbChirp.UserID {
		respondWithText(w, 403, "Permission denied")
		return
	}

	err = cfg.db.DeleteChirp(req.Context(), chirpID)
	if err != nil {
		fErr := fmt.Sprintf("Delete failed: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	respondWithText(w, 204, "Chirp deleted")
}
