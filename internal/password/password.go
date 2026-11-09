package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	minLength = 10
	argonTime = 1
	// 19 MiB fits api/worker 128M Compose limits on the 2 GB VPS (spec: single-thread, tight RAM).
	argonMem  = 19 * 1024
	argonThr  = 4
	argonKey  = 32
	argonSalt = 16
)

func Validate(pw string) error {
	if len(pw) < minLength {
		return fmt.Errorf("password must be at least %d characters", minLength)
	}
	return nil
}

func Hash(pw string) (string, error) {
	salt := make([]byte, argonSalt)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, argonTime, argonMem, argonThr, argonKey)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMem, argonTime, argonThr,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

func Compare(hash, pw string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("unsupported hash format")
	}
	var ver int
	var m, t, p uint32
	if _, err := fmt.Sscanf(parts[2], "v=%d", &ver); err != nil {
		return false, err
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(pw), salt, t, m, uint8(p), uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
