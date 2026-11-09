package version

import "testing"

func TestInfo(t *testing.T) {
	info := Info()
	if info["version"] == "" {
		t.Fatal("missing version")
	}
}
