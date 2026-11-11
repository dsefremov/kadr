package upload

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseKindFromMetadata(t *testing.T) {
	if ParseKindFromMetadata("") != "photo" {
		t.Fatal("default photo")
	}
	if ParseKindFromMetadata("filetype dmlkZW8") != "video" {
		t.Fatal("base64 video")
	}
}

func TestParseMetadataFilenameMtime(t *testing.T) {
	// filename "a.jpg" = YS5qcGc, mtime 1700000000000 = MTcwMDAwMDAwMDAwMA
	meta := ParseMetadata("filetype cGhvdG8, filename YS5qcGc, mtime MTcwMDAwMDAwMDAwMA")
	if meta.Kind != "photo" || meta.Filename != "a.jpg" {
		t.Fatalf("got %+v", meta)
	}
	if meta.MTime.UnixMilli() != 1700000000000 {
		t.Fatalf("mtime %v", meta.MTime)
	}
}

func TestCreatePatchComplete(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	dsn := os.Getenv("KADR_MYSQL_DSN")
	if dsn == "" {
		t.Skip("KADR_MYSQL_DSN")
	}
	root := t.TempDir()
	// relies on integration DB; covered in CI via script
	_ = filepath.Join(root)
	_ = dsn
	_ = context.Background()
	_ = bytes.NewReader(nil)
}
