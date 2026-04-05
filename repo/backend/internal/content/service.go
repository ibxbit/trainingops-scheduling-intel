package content

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	repo    *Repository
	storage *Storage
}

func NewService(repo *Repository, storage *Storage) *Service {
	return &Service{repo: repo, storage: storage}
}

func (s *Service) StartUpload(ctx context.Context, tenantID, userID string, documentID *string, fileName, mimeType string, totalChunks, chunkSize int) (*UploadSession, error) {
	if totalChunks <= 0 || chunkSize <= 0 {
		return nil, errors.New("invalid chunk settings")
	}
	return s.repo.CreateUploadSession(ctx, tenantID, userID, documentID, fileName, mimeType, totalChunks, chunkSize)
}

func (s *Service) UploadChunk(ctx context.Context, tenantID, uploadID string, chunkIndex int, body io.Reader, checksum string) error {
	us, err := s.repo.GetUploadSession(ctx, tenantID, uploadID)
	if err != nil {
		return err
	}
	if us.CompletedAt != nil {
		return errors.New("upload completed")
	}
	if time.Now().UTC().After(us.ExpiresAt) {
		return errors.New("upload expired")
	}
	if chunkIndex < 0 || chunkIndex >= us.TotalChunks {
		return errors.New("invalid chunk index")
	}

	path, n, sum, err := s.storage.SaveChunk(tenantID, uploadID, chunkIndex, body)
	if err != nil {
		return err
	}
	if checksum != "" && !strings.EqualFold(checksum, sum) {
		return errors.New("chunk checksum mismatch")
	}
	return s.repo.UpsertUploadChunk(ctx, tenantID, uploadID, chunkIndex, path, n, sum)
}

func (s *Service) CompleteUpload(ctx context.Context, tenantID, userID, uploadID string, title, summary string, difficulty, duration int) (*DocumentVersion, error) {
	if difficulty < 1 || difficulty > 5 {
		return nil, errors.New("difficulty must be 1-5")
	}
	if duration < 5 || duration > 480 {
		return nil, errors.New("duration must be 5-480")
	}

	us, err := s.repo.GetUploadSession(ctx, tenantID, uploadID)
	if err != nil {
		return nil, err
	}
	count, err := s.repo.CountUploadedChunks(ctx, tenantID, uploadID)
	if err != nil {
		return nil, err
	}
	if count != us.TotalChunks {
		return nil, errors.New("upload incomplete")
	}

	docID := ""
	if us.DocumentID != nil {
		docID = *us.DocumentID
	} else {
		docID = "temp"
	}
	storagePath, size, checksum, err := s.storage.AssembleUpload(tenantID, uploadID, docID, 1, us.FileName, us.TotalChunks)
	if err != nil {
		return nil, err
	}

	extracted := ""
	if strings.HasPrefix(us.MimeType, "text/") {
		f, err := s.storage.Open(storagePath)
		if err == nil {
			defer f.Close()
			buf := make([]byte, 8192)
			n, _ := f.Read(buf)
			extracted = string(buf[:n])
		}
	}

	v, err := s.repo.FinalizeUpload(ctx, tenantID, userID, uploadID, storagePath, size, checksum, extracted, title, summary, difficulty, duration)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) Preview(ctx context.Context, tenantID, documentID string, versionNo *int) (*DocumentVersion, io.ReadCloser, string, error) {
	v, err := s.repo.GetDocumentVersion(ctx, tenantID, documentID, versionNo)
	if err != nil {
		return nil, nil, "", err
	}
	if !isPreviewable(v.MimeType, v.FileName) {
		return nil, nil, "", errors.New("not previewable")
	}
	f, err := s.storage.Open(v.StoragePath)
	if err != nil {
		return nil, nil, "", err
	}
	return v, f, detectMime(v.MimeType, v.FileName), nil
}

func (s *Service) DownloadWatermarked(ctx context.Context, tenantID, documentID, actor string, versionNo *int) (*DocumentVersion, io.ReadCloser, string, error) {
	v, err := s.repo.GetDocumentVersion(ctx, tenantID, documentID, versionNo)
	if err != nil {
		return nil, nil, "", err
	}
	f, err := s.storage.Open(v.StoragePath)
	if err != nil {
		return nil, nil, "", err
	}
	m := detectMime(v.MimeType, v.FileName)
	wm := fmt.Sprintf("WATERMARK user=%s document=%s version=%d time=%s", actor, documentID, v.VersionNo, time.Now().UTC().Format(time.RFC3339))
	if strings.HasPrefix(m, "text/") {
		content, _ := io.ReadAll(f)
		_ = f.Close()
		payload := []byte(wm + "\n" + string(content))
		return v, io.NopCloser(strings.NewReader(string(payload))), m, nil
	}
	if m == "application/pdf" {
		content, _ := io.ReadAll(f)
		_ = f.Close()
		payload := append(content, []byte("\n% "+wm+"\n")...)
		return v, io.NopCloser(strings.NewReader(string(payload))), m, nil
	}
	return v, f, m, nil
}

func (s *Service) CreateShareLink(ctx context.Context, tenantID, userID, documentID string, versionNo *int) (string, time.Time, error) {
	var versionID *string
	if versionNo != nil {
		v, err := s.repo.GetDocumentVersion(ctx, tenantID, documentID, versionNo)
		if err != nil {
			return "", time.Time{}, err
		}
		versionID = &v.DocumentVersionID
	}
	raw, err := randomToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().UTC().Add(72 * time.Hour)
	_, err = s.repo.CreateShareLink(ctx, tenantID, userID, documentID, versionID, expires, raw)
	if err != nil {
		return "", time.Time{}, err
	}
	return raw, expires, nil
}

func (s *Service) ResolveShareDownload(ctx context.Context, rawToken string) (string, string, *int, error) {
	return s.repo.ResolveShareToken(ctx, rawToken)
}

func (s *Service) Search(ctx context.Context, tenantID, q string, limit int) ([]Document, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	return s.repo.SearchDocuments(ctx, tenantID, q, limit)
}

func (s *Service) Versions(ctx context.Context, tenantID, documentID string) ([]DocumentVersion, error) {
	return s.repo.ListVersions(ctx, tenantID, documentID)
}

func (s *Service) BulkUpdate(ctx context.Context, tenantID, actorUserID string, documentIDs, categoryIDs, tagIDs []string, archive *bool) error {
	return s.repo.BulkUpdate(ctx, tenantID, actorUserID, documentIDs, categoryIDs, tagIDs, archive)
}

func (s *Service) DetectDuplicates(ctx context.Context, tenantID, actorUserID string) (int, error) {
	return s.repo.DetectDuplicates(ctx, tenantID, actorUserID)
}

func (s *Service) SetMergeCandidate(ctx context.Context, tenantID, duplicateID string, candidate bool) error {
	return s.repo.SetMergeCandidate(ctx, tenantID, duplicateID, candidate)
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func isPreviewable(mimeType, fileName string) bool {
	m := detectMime(mimeType, fileName)
	if m == "application/pdf" {
		return true
	}
	if strings.HasPrefix(m, "image/") {
		return true
	}
	return strings.HasPrefix(m, "text/")
}

func detectMime(mimeType, fileName string) string {
	if mimeType != "" {
		return mimeType
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	if v := mime.TypeByExtension(ext); v != "" {
		return v
	}
	return "application/octet-stream"
}
