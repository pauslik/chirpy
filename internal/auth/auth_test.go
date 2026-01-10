package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHahsing(t *testing.T) {
	pass := "password"

	hash, err := HashPassword(pass)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if len(hash) < 80 {
		t.Errorf("Bad hash: %s\n", hash)
		t.Fail()
	}

	tf, err := CheckPasswordHash(pass, hash)

	if !tf {
		t.Errorf("Hash check failed\n")
		t.Fail()
	}
}

func TestBearer(t *testing.T) {
	bad := "OfBadNews"
	bear := fmt.Sprintf("Bearer %s", bad)
	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	req.Header.Set("Authorization", bear)

	token, err := GetBearerToken(req.Header)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if strings.Compare(bad, token) != 0 {
		fmt.Printf("Bearer mismatch: %s vs %s\n", bad, token)
		t.Fail()
	}
}
