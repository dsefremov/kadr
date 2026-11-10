package feed

import "testing"

func TestValidVisibility(t *testing.T) {
	for _, tc := range []struct {
		vis string
		ok  bool
	}{
		{"public", true},
		{"private", true},
		{"hidden", false},
		{"", false},
	} {
		if ValidVisibility(tc.vis) != tc.ok {
			t.Fatalf("%q want %v", tc.vis, tc.ok)
		}
	}
}
