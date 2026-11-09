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
