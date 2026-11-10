package feed

import "testing"

func TestBuildPhotoURLs(t *testing.T) {
	urls := buildURLs("abc123", "photo")
	if urls["preview_320"] == "" || urls["preview_1280"] == "" {
		t.Fatalf("previews: %v", urls)
	}
	if urls["preview_640"] != "/m/abc123/preview-640.v1sdr20261111" {
		t.Fatalf("640: %s", urls["preview_640"])
	}
}
