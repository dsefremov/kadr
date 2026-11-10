package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func RandomURLSafe(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func Hash(plain string) [32]byte {
	return sha256.Sum256([]byte(plain))
}
