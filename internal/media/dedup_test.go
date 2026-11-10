package media

import (
	"testing"
)

func TestDedupConstants(t *testing.T) {
	if ProfileHashV1SDR != "v1sdr20261111" {
		t.Fatalf("profile hash: %s", ProfileHashV1SDR)
	}
	if VariantPreview320 != "preview-320" || VariantPreview1280 != "preview-1280" {
		t.Fatal("preview variant names")
	}
}
