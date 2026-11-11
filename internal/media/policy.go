package media

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dsefremov/kadr/internal/units"
)

// SniffKind returns photo, video, or empty if unsupported.
// Incoming tus files often have no extension — sniff magic bytes too.
func SniffKind(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".heic", ".heif":
		return "photo", nil
	case ".mp4", ".mov", ".m4v", ".mkv", ".webm":
		return "video", nil
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	head := make([]byte, 16)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	if bytes.HasPrefix(head, []byte{0xff, 0xd8}) {
		return "photo", nil
	}
	if bytes.HasPrefix(head, []byte{0x89, 'P', 'N', 'G'}) {
		return "photo", nil
	}
	if len(head) >= 12 && string(head[8:12]) == "webp" {
		return "photo", nil
	}
	// ISO BMFF (MP4 / MOV / M4V)
	if bytes.Contains(head, []byte("ftyp")) {
		return "video", nil
	}
	// Matroska / WebM (EBML)
	if bytes.HasPrefix(head, []byte{0x1a, 0x45, 0xdf, 0xa3}) {
		return "video", nil
	}
	return "", fmt.Errorf("%w: extension %s", ErrUnsupported, ext)
}

// IsAnimatedImage detects animated GIF/WebP/PNG (APNG).
func IsAnimatedImage(path string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(path))
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	switch ext {
	case ".gif":
		return bytes.Contains(data, []byte("NETSCAPE2.0")), nil
	case ".webp":
		return bytes.Contains(data, []byte("ANIM")), nil
	}
	if bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}) {
		if bytes.Contains(data, []byte("acTL")) {
			return true, nil
		}
	}
	if ext == ".webp" || (len(data) >= 12 && string(data[8:12]) == "webp") {
		// RIFF....WEBP with ANIM chunk
		return scanWebPAnimated(data), nil
	}
	return false, nil
}

func scanWebPAnimated(data []byte) bool {
	return bytes.Contains(data, []byte("ANIM"))
}

// CheckDurationMs rejects videos over spec limit.
func CheckDurationMs(ms uint64) error {
	if ms > uint64(units.VideoMaxDurationMs) {
		return ErrVideoTooLong
	}
	return nil
}

// ParseDurationToMs parses ffprobe duration seconds string edge cases in tests.
func ParseDurationToMs(seconds float64) uint64 {
	if seconds < 0 {
		return 0
	}
	return uint64(seconds * 1000)
}

// IsAPNG reports acTL chunk in PNG (for unit tests with minimal buffer).
func IsAPNG(data []byte) bool {
	if !bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}) {
		return false
	}
	return bytes.Contains(data, []byte("acTL"))
}

// PNGHasAlpha is a heuristic for tests (tRNS or RGBA color type).
func PNGHasAlpha(data []byte) bool {
	if len(data) < 26 || !bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}) {
		return false
	}
	// IHDR color type at offset 25
	if data[25] == 4 || data[25] == 6 {
		return true
	}
	return bytes.Contains(data, []byte("tRNS"))
}

// ReadPNGColorType for tests.
func ReadPNGColorType(data []byte) int {
	if len(data) < 26 {
		return -1
	}
	return int(data[25])
}

// ScaleToFit3200 returns target long-side cap semantics (no upscale).
func ScaleToFit3200(w, h int) (int, int) {
	const cap = 3200
	long := w
	if h > long {
		long = h
	}
	if long <= cap {
		return w, h
	}
	scale := float64(cap) / float64(long)
	return int(float64(w) * scale), int(float64(h) * scale)
}

// FlattenSortKey builds a deterministic key fragment from dimensions for tests.
func FlattenSortKey(v uint64) uint64 {
	return binary.LittleEndian.Uint64(bytes.TrimLeft(binary.AppendUvarint(nil, v), "\x00"))
}
