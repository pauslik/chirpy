package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	// Add checks for password complexity here
	params := &argon2id.Params{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
	return argon2id.CreateHash(password, params)
}

func CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func GetBearerToken(headers http.Header) (string, error) {
	bear := headers.Get("Authorization")
	split := strings.Split(bear, " ")
	if len(split) != 2 {
		return "", errors.New("Wrong bearer provided.\n")
	}

	return split[1], nil

}
