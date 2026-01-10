package auth

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestJWT(t *testing.T) {
	id, _ := uuid.Parse("52ad359f-88b7-419f-9be0-6632062d5e3e")
	secret := "TokenSecret"

	jwtStr, err := MakeJWT(id, secret)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if len(jwtStr) < 100 {
		t.Errorf("Bad secret: %s\n", jwtStr)
		t.Fail()
	}

	id2, err := ValidateJWT(jwtStr, secret)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if strings.Compare(fmt.Sprintf("%v", id), fmt.Sprintf("%v", id2)) != 0 {
		t.Errorf("Validation failed: %v vs %v\n", id, id2)
		t.Fail()
	}
}
