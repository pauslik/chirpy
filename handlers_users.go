package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pauslik/chirpy/internal/auth"
	"github.com/pauslik/chirpy/internal/database"
)

type inputUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type outputUser struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

type outputRefresToken struct {
	Token string `json:"token"`
}

func (cfg *apiConfig) usersHandler(w http.ResponseWriter, req *http.Request) {
	iUser := inputUser{}
	oUser := outputUser{}
	// Decoding input
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&iUser)
	if err != nil {
		fErr := fmt.Sprintf("Error decoding input: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	passHashed, err := auth.HashPassword(iUser.Password)
	if err != nil {
		fErr := fmt.Sprintf("Error hashing password: %s", err)
		respondWithText(w, 500, fErr)
		return
	}
	cup := database.CreateUserParams{
		Email:          iUser.Email,
		HashedPassword: passHashed,
	}
	dbUser, err := cfg.db.CreateUser(req.Context(), cup)
	if err != nil {
		fErr := fmt.Sprintf("Error creating user: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	oUser.Email = dbUser.Email
	oUser.ID = dbUser.ID
	oUser.CreatedAt = dbUser.CreatedAt
	oUser.UpdatedAt = dbUser.UpdatedAt

	respondWithJSON(w, 201, oUser)
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, req *http.Request) {
	iUser := inputUser{}
	oUser := outputUser{}
	// Decoding input
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&iUser)
	if err != nil {
		fErr := fmt.Sprintf("Error decoding input: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	// Check if user exists
	dbUser, err := cfg.db.GetUser(req.Context(), iUser.Email)
	if err != nil {
		// Handle not found error
		if errors.Is(err, sql.ErrNoRows) {
			respondWithText(w, 401, "Incorrect email or password")
			return
		}
		fErr := fmt.Sprintf("Error fetching user by email: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	// Check if password matches
	correct, err := auth.CheckPasswordHash(iUser.Password, dbUser.HashedPassword)
	if err != nil {
		fErr := fmt.Sprintf("Error checking password: %s", err)
		respondWithText(w, 500, fErr)
		return
	}
	if !correct {
		respondWithText(w, 401, "Incorrect email or password")
	}

	token, err := auth.MakeJWT(dbUser.ID, cfg.jwt)
	if err != nil {
		fErr := fmt.Sprintf("Could not generate JWT token: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	oUser.Email = dbUser.Email
	oUser.ID = dbUser.ID
	oUser.CreatedAt = dbUser.CreatedAt
	oUser.UpdatedAt = dbUser.UpdatedAt
	oUser.Token = token
	// create refresh token and save it in database, maybe there's a better way to do this?
	refreshToken, _ := auth.MakeRefreshToken()
	crtp := database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    oUser.ID,
		ExpiresAt: time.Now().AddDate(0, 0, 60),
	}
	dbRefreshToken, err := cfg.db.CreateRefreshToken(req.Context(), crtp)
	if err != nil {
		fErr := fmt.Sprintf("Could not create refresh token: %s", err)
		respondWithText(w, 500, fErr)
		return
	}
	oUser.RefreshToken = dbRefreshToken.Token

	respondWithJSON(w, 200, oUser)
}

func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, req *http.Request) {
	oToken := outputRefresToken{}

	bearer, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fErr := fmt.Sprintf("Error getting bearer: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	dbRefreshToken, err := cfg.db.GetRefreshToken(req.Context(), bearer)
	if err != nil {
		// Handle not found error
		if errors.Is(err, sql.ErrNoRows) {
			respondWithText(w, 401, "Refresh token not found or expired")
			return
		}
		fErr := fmt.Sprintf("Error getting chirps: %s", err)
		respondWithText(w, 500, fErr)
		return
	}
	// Handle Null value
	if !dbRefreshToken.RevokedAt.Time.IsZero() {
		respondWithText(w, 401, "Refresh token not found or expired")
		return
	}

	// GetUserFromRefreshToken
	dbUser, err := cfg.db.GetUserFromRefreshToken(req.Context(), bearer)
	if err != nil {
		fErr := fmt.Sprintf("No user associated with refresh token: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	// Create new JWT
	token, err := auth.MakeJWT(dbUser.ID, cfg.jwt)
	if err != nil {
		fErr := fmt.Sprintf("Could not generate JWT token: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	oToken.Token = token

	respondWithJSON(w, 200, oToken)
}

func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, req *http.Request) {
	bearer, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fErr := fmt.Sprintf("Error getting bearer: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	// Revoke the token, setting the time to current timestamp is done in database, is that ok?
	cfg.db.RevokeRefreshToken(req.Context(), bearer)

	respondWithText(w, 204, "")
}
