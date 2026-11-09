package auth

import "testing"

func TestErrInvalidTokenReset(t *testing.T) {
	if errInvalidToken().Error() != "invalid token" {
		t.Fatal()
	}
}
