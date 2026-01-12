package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/pauslik/chirpy/internal/auth"
)

type inputUpgrade struct {
	Event string `json:"event"`
	Data  struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

func (cfg *apiConfig) upgradeRedHandler(w http.ResponseWriter, req *http.Request) {
	iUpgrade := inputUpgrade{}

	// Check if API key matches the expected
	apiKey, _ := auth.GetAPIKey(req.Header)
	if apiKey != cfg.polka {
		respondWithText(w, 401, "Wrong API key")
		return
	}

	// Decoding input
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&iUpgrade)
	if err != nil {
		fErr := fmt.Sprintf("Error decoding input: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	if iUpgrade.Event != "user.upgraded" {
		respondWithText(w, 204, "")
		return
	}

	parsedID, err := uuid.Parse(iUpgrade.Data.UserID)
	if err != nil {
		fErr := fmt.Sprintf("Not valid user ID: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	_, err = cfg.db.UpgradeUserRed(req.Context(), parsedID)
	if err != nil {
		fErr := fmt.Sprintf("User not found with respective ID: %s", err)
		respondWithText(w, 500, fErr)
		return
	}

	respondWithText(w, 204, "")
}
