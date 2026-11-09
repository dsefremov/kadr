package admin

import "testing"

func TestAuthSanitize(t *testing.T) {
	if got := authSanitize("Alice.Bob"); got != "alicebob" {
		t.Fatalf("got %q", got)
	}
}
