package content

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"trainingops/backend/internal/dbctx"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrVersionConflict = errors.New("version conflict")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUploadSession(ctx context.Context, tenantID, userID string, documentID *string, fileName, mimeType string, totalChunks, chunkSize int) (*UploadSession, error) {
	var s UploadSession
	var doc sql.NullString
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO upload_sessions (
  tenant_id, document_id, file_name, mime_type, total_chunks, chunk_size_bytes, expires_at, created_by_user_id
)
VALUES ($1::uuid, NULLIF($2, '')::uuid, $3, $4, $5, $6, NOW() + INTERVAL '72 hours', $7::uuid)
RETURNING upload_id::text, document_id::text, file_name, mime_type, total_chunks, chunk_size_bytes, expires_at, completed_at
`, tenantID, nullable(documentID), fileName, mimeType, totalChunks, chunkSize, userID).Scan(
		&s.UploadID,
		&doc,
		&s.FileName,
		&s.MimeType,
		&s.TotalChunks,
		&s.ChunkSizeBytes,
		&s.ExpiresAt,
		&s.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	if doc.Valid {
		s.DocumentID = &doc.String
	}
	return &s, nil
}

func (r *Repository) GetUploadSession(ctx context.Context, tenantID, uploadID string) (*UploadSession, error) {
	var s UploadSession
	var doc sql.NullString
	err := dbctx.QueryRowContext(ctx, r.db, `
SELECT upload_id::text, document_id::text, file_name, mime_type, total_chunks, chunk_size_bytes, expires_at, completed_at
FROM upload_sessions
WHERE tenant_id::text = $1 AND upload_id::text = $2
`, tenantID, uploadID).Scan(
		&s.UploadID,
		&doc,
		&s.FileName,
		&s.MimeType,
		&s.TotalChunks,
		&s.ChunkSizeBytes,
		&s.ExpiresAt,
		&s.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if doc.Valid {
		s.DocumentID = &doc.String
	}
	return &s, nil
}

func (r *Repository) UpsertUploadChunk(ctx context.Context, tenantID, uploadID string, chunkIndex int, storagePath string, chunkSize int64, checksum string) error {
	_, err := dbctx.ExecContext(ctx, r.db, `
INSERT INTO upload_chunks (tenant_id, upload_id, chunk_index, storage_path, chunk_size_bytes, sha256_checksum)
VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
ON CONFLICT (tenant_id, upload_id, chunk_index)
DO UPDATE SET
  storage_path = EXCLUDED.storage_path,
  chunk_size_bytes = EXCLUDED.chunk_size_bytes,
  sha256_checksum = EXCLUDED.sha256_checksum,
  received_at = NOW()
`, tenantID, uploadID, chunkIndex, storagePath, chunkSize, checksum)
	return err
}

func (r *Repository) CountUploadedChunks(ctx context.Context, tenantID, uploadID string) (int, error) {
	var n int
	err := dbctx.QueryRowContext(ctx, r.db, `
SELECT COUNT(*)
FROM upload_chunks
WHERE tenant_id::text = $1 AND upload_id::text = $2
`, tenantID, uploadID).Scan(&n)
	return n, err
}

func (r *Repository) FinalizeUpload(ctx context.Context, tenantID, userID, uploadID, storagePath string, fileSize int64, checksum string, extractedText string, title, summary string, difficulty, duration int) (*DocumentVersion, error) {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var s UploadSession
	var doc sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT upload_id::text, document_id::text, file_name, mime_type, total_chunks, chunk_size_bytes, expires_at, completed_at
FROM upload_sessions
WHERE tenant_id::text = $1 AND upload_id::text = $2
FOR UPDATE
`, tenantID, uploadID).Scan(
		&s.UploadID,
		&doc,
		&s.FileName,
		&s.MimeType,
		&s.TotalChunks,
		&s.ChunkSizeBytes,
		&s.ExpiresAt,
		&s.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if doc.Valid {
		s.DocumentID = &doc.String
	}
	if s.CompletedAt != nil {
		return nil, errors.New("upload already completed")
	}

	var chunkCount int
	err = tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM upload_chunks
WHERE tenant_id::text = $1 AND upload_id::text = $2
`, tenantID, uploadID).Scan(&chunkCount)
	if err != nil {
		return nil, err
	}
	if chunkCount != s.TotalChunks {
		return nil, errors.New("incomplete upload")
	}

	documentID := ""
	resolvedTitle := strings.TrimSpace(title)
	if resolvedTitle == "" {
		resolvedTitle = s.FileName
	}
	resolvedSummary := strings.TrimSpace(summary)
	resolvedDifficulty := difficulty
	if resolvedDifficulty < 1 || resolvedDifficulty > 5 {
		resolvedDifficulty = 1
	}
	resolvedDuration := duration
	if resolvedDuration < 5 || resolvedDuration > 480 {
		resolvedDuration = 5
	}
	if s.DocumentID == nil {
		err = tx.QueryRowContext(ctx, `
INSERT INTO documents (tenant_id, title, summary, difficulty, duration_minutes, created_by_user_id, updated_by_user_id)
VALUES ($1::uuid, $2, $3, $4, $5, $6::uuid, $6::uuid)
RETURNING document_id::text
`, tenantID, resolvedTitle, resolvedSummary, resolvedDifficulty, resolvedDuration, userID).Scan(&documentID)
		if err != nil {
			return nil, err
		}
	} else {
		documentID = *s.DocumentID
		if _, err := tx.ExecContext(ctx, `
UPDATE documents
SET title = $3,
    summary = $4,
    difficulty = $5,
    duration_minutes = $6,
    updated_by_user_id = $7::uuid,
    updated_at = NOW()
WHERE tenant_id::text = $1 AND document_id::text = $2
`, tenantID, documentID, resolvedTitle, resolvedSummary, resolvedDifficulty, resolvedDuration, userID); err != nil {
			return nil, err
		}
	}

	var nextVersion int
	err = tx.QueryRowContext(ctx, `
UPDATE documents
SET current_version_no = current_version_no + 1,
    updated_by_user_id = $3::uuid,
    updated_at = NOW()
WHERE tenant_id::text = $1 AND document_id::text = $2
RETURNING current_version_no
`, tenantID, documentID, userID).Scan(&nextVersion)
	if err != nil {
		return nil, err
	}

	v := &DocumentVersion{}
	err = tx.QueryRowContext(ctx, `
INSERT INTO document_versions (
  tenant_id, document_id, version_no, file_name, storage_path, mime_type,
  file_size_bytes, sha256_checksum, extracted_text, created_by_user_id
)
VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10::uuid)
RETURNING document_version_id::text, document_id::text, version_no, file_name, storage_path, mime_type, file_size_bytes, sha256_checksum, created_at
`, tenantID, documentID, nextVersion, s.FileName, storagePath, s.MimeType, fileSize, checksum, extractedText, userID).Scan(
		&v.DocumentVersionID,
		&v.DocumentID,
		&v.VersionNo,
		&v.FileName,
		&v.StoragePath,
		&v.MimeType,
		&v.FileSizeBytes,
		&v.SHA256Checksum,
		&v.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
UPDATE upload_sessions
SET completed_at = NOW(), document_id = $3::uuid
WHERE tenant_id::text = $1 AND upload_id::text = $2
`, tenantID, uploadID, documentID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return v, nil
}

func (r *Repository) GetDocumentVersion(ctx context.Context, tenantID, documentID string, versionNo *int) (*DocumentVersion, error) {
	q := `
SELECT dv.document_version_id::text, dv.document_id::text, dv.version_no, dv.file_name, dv.storage_path, dv.mime_type, dv.file_size_bytes, dv.sha256_checksum, dv.created_at
FROM document_versions dv
JOIN documents d ON d.tenant_id = dv.tenant_id AND d.document_id = dv.document_id
WHERE dv.tenant_id::text = $1 AND dv.document_id::text = $2`
	args := []any{tenantID, documentID}
	if versionNo == nil {
		q += ` AND dv.version_no = d.current_version_no`
	} else {
		q += ` AND dv.version_no = $3`
		args = append(args, *versionNo)
	}

	v := &DocumentVersion{}
	err := dbctx.QueryRowContext(ctx, r.db, q, args...).Scan(
		&v.DocumentVersionID,
		&v.DocumentID,
		&v.VersionNo,
		&v.FileName,
		&v.StoragePath,
		&v.MimeType,
		&v.FileSizeBytes,
		&v.SHA256Checksum,
		&v.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return v, nil
}

func (r *Repository) ListVersions(ctx context.Context, tenantID, documentID string) ([]DocumentVersion, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT document_version_id::text, document_id::text, version_no, file_name, storage_path, mime_type, file_size_bytes, sha256_checksum, created_at
FROM document_versions
WHERE tenant_id::text = $1 AND document_id::text = $2
ORDER BY version_no DESC
`, tenantID, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DocumentVersion, 0)
	for rows.Next() {
		var v DocumentVersion
		if err := rows.Scan(&v.DocumentVersionID, &v.DocumentID, &v.VersionNo, &v.FileName, &v.StoragePath, &v.MimeType, &v.FileSizeBytes, &v.SHA256Checksum, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) SearchDocuments(ctx context.Context, tenantID, q string, limit int) ([]Document, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT document_id::text, tenant_id::text, title, coalesce(summary, ''), difficulty, duration_minutes, current_version_no, is_archived
FROM documents
WHERE tenant_id::text = $1
  AND (search_vector @@ plainto_tsquery('simple', $2)
       OR EXISTS (
         SELECT 1 FROM document_versions dv
         WHERE dv.tenant_id = documents.tenant_id
           AND dv.document_id = documents.document_id
           AND dv.search_vector @@ plainto_tsquery('simple', $2)
       ))
ORDER BY updated_at DESC
LIMIT $3
`, tenantID, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Document, 0)
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.DocumentID, &d.TenantID, &d.Title, &d.Summary, &d.Difficulty, &d.DurationMinutes, &d.CurrentVersionNo, &d.IsArchived); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) CreateShareLink(ctx context.Context, tenantID, userID, documentID string, versionID *string, expiresAt time.Time, rawToken string) (string, error) {
	tokenHash := hashToken(rawToken)
	var id string
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO document_share_links (tenant_id, document_id, document_version_id, token_hash, expires_at, created_by_user_id)
VALUES ($1::uuid, $2::uuid, NULLIF($3, '')::uuid, $4, $5, $6::uuid)
RETURNING share_link_id::text
`, tenantID, documentID, nullable(versionID), tokenHash, expiresAt, userID).Scan(&id)
	return id, err
}

func (r *Repository) ResolveShareToken(ctx context.Context, rawToken string) (tenantID, documentID string, versionNo *int, err error) {
	h := hashToken(rawToken)
	var v sql.NullInt32
	err = dbctx.QueryRowContext(ctx, r.db, `
SELECT sl.tenant_id::text, sl.document_id::text, dv.version_no
FROM document_share_links sl
LEFT JOIN document_versions dv ON dv.tenant_id = sl.tenant_id AND dv.document_version_id = sl.document_version_id
WHERE sl.token_hash = $1
  AND sl.expires_at > NOW()
`, h).Scan(&tenantID, &documentID, &v)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", nil, ErrNotFound
		}
		return "", "", nil, err
	}
	if v.Valid {
		t := int(v.Int32)
		versionNo = &t
	}
	_, _ = dbctx.ExecContext(ctx, r.db, `UPDATE document_share_links SET used_count = used_count + 1 WHERE token_hash = $1`, h)
	return tenantID, documentID, versionNo, nil
}

func (r *Repository) BulkUpdate(ctx context.Context, tenantID, actorUserID string, documentIDs, addCategoryIDs, addTagIDs []string, archive *bool) error {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, docID := range documentIDs {
		if archive != nil {
			_, err := tx.ExecContext(ctx, `
UPDATE documents
SET is_archived = $3, updated_by_user_id = $4::uuid, updated_at = NOW()
WHERE tenant_id::text = $1 AND document_id::text = $2
`, tenantID, docID, *archive, actorUserID)
			if err != nil {
				return err
			}
		}
		for _, cID := range addCategoryIDs {
			_, err := tx.ExecContext(ctx, `
INSERT INTO document_categories (tenant_id, document_id, category_id)
VALUES ($1::uuid, $2::uuid, $3::uuid)
ON CONFLICT DO NOTHING
`, tenantID, docID, cID)
			if err != nil {
				return err
			}
		}
		for _, tID := range addTagIDs {
			_, err := tx.ExecContext(ctx, `
INSERT INTO document_tags (tenant_id, document_id, tag_id)
VALUES ($1::uuid, $2::uuid, $3::uuid)
ON CONFLICT DO NOTHING
`, tenantID, docID, tID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *Repository) DetectDuplicates(ctx context.Context, tenantID, actorUserID string) (int, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT dv.document_id::text, dv.sha256_checksum
FROM document_versions dv
JOIN documents d ON d.tenant_id = dv.tenant_id AND d.document_id = dv.document_id AND d.current_version_no = dv.version_no
WHERE dv.tenant_id::text = $1
`, tenantID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	bySum := map[string][]string{}
	for rows.Next() {
		var docID, sum string
		if err := rows.Scan(&docID, &sum); err != nil {
			return 0, err
		}
		bySum[sum] = append(bySum[sum], docID)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	flagged := 0
	for sum, ids := range bySum {
		if len(ids) < 2 {
			continue
		}
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				left, right := ids[i], ids[j]
				if strings.Compare(left, right) > 0 {
					left, right = right, left
				}
				_, err := dbctx.ExecContext(ctx, r.db, `
INSERT INTO document_duplicate_flags (tenant_id, left_document_id, right_document_id, checksum, merge_candidate, flagged_by_user_id)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, FALSE, $5::uuid)
ON CONFLICT DO NOTHING
`, tenantID, left, right, sum, actorUserID)
				if err != nil {
					return flagged, err
				}
				flagged++
			}
		}
	}
	return flagged, nil
}

func (r *Repository) SetMergeCandidate(ctx context.Context, tenantID, duplicateID string, candidate bool) error {
	res, err := dbctx.ExecContext(ctx, r.db, `
UPDATE document_duplicate_flags
SET merge_candidate = $3
WHERE tenant_id::text = $1 AND duplicate_id::text = $2
`, tenantID, duplicateID, candidate)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func nullable(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func hashToken(raw string) string {
	s := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(s[:])
}
