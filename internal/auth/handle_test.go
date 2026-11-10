package auth

import "testing"

func TestSanitizeHandle(t *testing.T) {
	if sanitizeHandle("A.B+C") != "abc" {
		t.Fatal(sanitizeHandle("A.B+C"))
	}
	if sanitizeHandle("!!") != "user" {
		t.Fatal("fallback")
	}
}
