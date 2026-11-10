package media

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/dsefremov/kadr/internal/units"
)

func TestIsAPNG(t *testing.T) {
	static := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0, 'I', 'H', 'D', 'R'}
	if IsAPNG(static) {
		t.Fatal("static png should not be animated")
	}
	anim := append(bytes.Clone(static), []byte("acTL")...)
	if !IsAPNG(anim) {
		t.Fatal("expected apng")
	}
}

func TestScaleToFit3200(t *testing.T) {
	w, h := ScaleToFit3200(4000, 3000)
	if w > 3200 || h > 3200 {
		t.Fatalf("expected fit in 3200, got %dx%d", w, h)
	}
	w2, h2 := ScaleToFit3200(800, 600)
	if w2 != 800 || h2 != 600 {
		t.Fatalf("no upscale expected, got %dx%d", w2, h2)
	}
}

func TestCheckDurationMs(t *testing.T) {
	if err := CheckDurationMs(uint64(units.VideoMaxDurationMs)); err != nil {
		t.Fatal(err)
	}
	if err := CheckDurationMs(uint64(units.VideoMaxDurationMs) + 1); err != ErrVideoTooLong {
		t.Fatalf("want ErrVideoTooLong, got %v", err)
	}
}

func TestSniffKindFixtures(t *testing.T) {
	root := filepath.Join("..", "..", "fixtures", "in")
	for _, name := range []string{"sample.png", "sample.heic"} {
		p := filepath.Join(root, name)
		if _, err := os.Stat(p); err != nil {
			t.Skip("fixtures missing")
		}
		k, err := SniffKind(p)
		if err != nil || k != "photo" {
			t.Fatalf("%s: %q %v", name, k, err)
		}
	}
	mov := filepath.Join(root, "sample.mov")
	if _, err := os.Stat(mov); err == nil {
		k, err := SniffKind(mov)
		if err != nil || k != "video" {
			t.Fatalf("mov: %q %v", k, err)
		}
	}
}

func TestAnimatedRejectsGIF(t *testing.T) {
	// minimal GIF with animation extension
	gif := []byte("GIF89a\x00\x00\x00\x00!\xff\x0bNETSCAPE2.0")
	tmp := t.TempDir() + "/a.gif"
	if err := os.WriteFile(tmp, gif, 0o644); err != nil {
		t.Fatal(err)
	}
	ok, err := IsAnimatedImage(tmp)
	if err != nil || !ok {
		t.Fatalf("gif animated: %v %v", ok, err)
	}
}
