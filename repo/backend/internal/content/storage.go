package content

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Storage struct {
	root string
}

func NewStorage(root string) *Storage {
	return &Storage{root: root}
}

func (s *Storage) SaveChunk(tenantID, uploadID string, chunkIndex int, src io.Reader) (string, int64, string, error) {
	dir := filepath.Join(s.root, "uploads", tenantID, uploadID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, "", err
	}
	path := filepath.Join(dir, strconv.Itoa(chunkIndex)+".part")
	f, err := os.Create(path)
	if err != nil {
		return "", 0, "", err
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), src)
	if err != nil {
		return "", 0, "", err
	}
	return path, n, hex.EncodeToString(h.Sum(nil)), nil
}

func (s *Storage) AssembleUpload(tenantID, uploadID, documentID string, versionNo int, fileName string, totalChunks int) (string, int64, string, error) {
	dstDir := filepath.Join(s.root, "documents", tenantID, documentID, strconv.Itoa(versionNo))
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return "", 0, "", err
	}
	dstPath := filepath.Join(dstDir, filepath.Base(fileName))
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", 0, "", err
	}
	defer dst.Close()

	h := sha256.New()
	var total int64
	for i := 0; i < totalChunks; i++ {
		partPath := filepath.Join(s.root, "uploads", tenantID, uploadID, strconv.Itoa(i)+".part")
		part, err := os.Open(partPath)
		if err != nil {
			return "", 0, "", err
		}
		n, copyErr := io.Copy(io.MultiWriter(dst, h), part)
		part.Close()
		if copyErr != nil {
			return "", 0, "", copyErr
		}
		total += n
	}
	return dstPath, total, hex.EncodeToString(h.Sum(nil)), nil
}

func (s *Storage) Open(path string) (*os.File, error) {
	cleanRoot := filepath.Clean(s.root)
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, cleanRoot) {
		return nil, errors.New("invalid storage path")
	}
	return os.Open(cleanPath)
}

func (s *Storage) SaveIngestedText(tenantID, sourceID, externalID string, body []byte) (string, int64, string, error) {
	dir := filepath.Join(s.root, "ingestion", tenantID, sourceID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, "", err
	}
	name := sanitizeStorageName(externalID)
	if name == "" {
		name = "item"
	}
	path := filepath.Join(dir, name+".txt")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", 0, "", err
	}
	h := sha256.Sum256(body)
	return path, int64(len(body)), hex.EncodeToString(h[:]), nil
}

func sanitizeStorageName(v string) string {
	b := make([]rune, 0, len(v))
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b = append(b, r)
		} else if r == ' ' {
			b = append(b, '_')
		}
	}
	return string(b)
}
