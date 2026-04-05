package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorageOpenRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	s := NewStorage(root)

	outside := filepath.Join(root, "..", "outside.txt")
	if err := os.WriteFile(outside, []byte("bad"), 0o644); err != nil {
		t.Fatalf("failed to seed outside file: %v", err)
	}

	_, err := s.Open(outside)
	if err == nil {
		t.Fatal("expected invalid storage path error")
	}
}

func TestSanitizeStorageName(t *testing.T) {
	got := sanitizeStorageName("../../Quarterly Report v1.pdf")
	if got == "" {
		t.Fatal("expected sanitized storage name")
	}
	if got == "../../Quarterly Report v1.pdf" {
		t.Fatal("expected sanitization to remove path controls")
	}
}
