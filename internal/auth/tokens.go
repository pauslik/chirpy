package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	TokenTypeAccess TokenType = "chirpy-access"
)

// JSON Web Token

func MakeJWT(userID uuid.UUID, tokenSecret string) (string, error) {
	utcTime := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    string(TokenTypeAccess),
		IssuedAt:  jwt.NewNumericDate(utcTime),
		ExpiresAt: jwt.NewNumericDate(utcTime.Add(time.Hour)),
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return signed, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claimsStruct := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claimsStruct, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	if !token.Valid {
		return uuid.Nil, errors.New("Invalid token")
	}

	userID := claimsStruct.Subject
	issuer := claimsStruct.Issuer

	if issuer != string(TokenTypeAccess) {
		return uuid.Nil, errors.New("Invalid issuer")
	}

	id, err := uuid.Parse(userID)

	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

// Refresh token

// do we need to return error? none of the functions used inside do
func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	rand.Read(key)
	encKey := hex.EncodeToString(key)
	return encKey, nil
}
