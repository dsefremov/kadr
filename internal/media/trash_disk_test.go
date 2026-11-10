package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoredDirRemovedFromDisk(t *testing.T) {
	root := t.TempDir()
	pub := "testpub"
	dir := filepath.Join(root, "stored", "42", pub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(dir, "master.jpg")
	if err := os.WriteFile(f, []byte("jpeg"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Fatalf("file should be gone: %v", err)
	}
}
