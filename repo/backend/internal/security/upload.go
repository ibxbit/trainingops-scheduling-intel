package security

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
)

var mimeByExt = map[string]string{
	"pdf":  "application/pdf",
	"txt":  "text/plain; charset=utf-8",
	"md":   "text/plain; charset=utf-8",
	"docx": "application/zip",
}

func ValidateUpload(fileHeader *multipart.FileHeader, allowed map[string]struct{}, expectedChecksum string) (string, error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileHeader.Filename)), ".")
	if _, ok := allowed[ext]; !ok {
		return "", errors.New("file format not allowed")
	}

	f, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	head := make([]byte, 512)
	n, err := f.Read(head)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	detected := mimeType(head[:n])
	allowedMime := mimeByExt[ext]
	if detected != allowedMime && !(ext == "md" && strings.HasPrefix(detected, "text/plain")) {
		return "", errors.New("file content type mismatch")
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if expectedChecksum != "" && !strings.EqualFold(expectedChecksum, actual) {
		return "", errors.New("checksum verification failed")
	}
	return actual, nil
}

func mimeType(b []byte) string {
	if len(b) >= 4 && b[0] == 0x25 && b[1] == 0x50 && b[2] == 0x44 && b[3] == 0x46 {
		return "application/pdf"
	}
	if len(b) >= 4 && b[0] == 0x50 && b[1] == 0x4b {
		return "application/zip"
	}
	return "text/plain; charset=utf-8"
}
