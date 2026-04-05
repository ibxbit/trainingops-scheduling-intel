package content

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"trainingops/backend/internal/access"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type startUploadRequest struct {
	DocumentID     *string `json:"document_id"`
	FileName       string  `json:"file_name"`
	MimeType       string  `json:"mime_type"`
	TotalChunks    int     `json:"total_chunks"`
	ChunkSizeBytes int     `json:"chunk_size_bytes"`
}

func (h *Handler) StartUpload(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req startUploadRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	s, err := h.svc.StartUpload(c.Request().Context(), tenantID, userID, req.DocumentID, req.FileName, req.MimeType, req.TotalChunks, req.ChunkSizeBytes)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": s})
}

func (h *Handler) UploadChunk(c echo.Context) error {
	tenantID, _, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	uploadID := c.Param("upload_id")
	chunkIndexRaw := c.Param("chunk_index")
	chunkIndex, err := strconv.Atoi(chunkIndexRaw)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid chunk_index"})
	}
	checksum := c.Request().Header.Get("X-Chunk-Checksum")
	if err := h.svc.UploadChunk(c.Request().Context(), tenantID, uploadID, chunkIndex, c.Request().Body, checksum); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "chunk_received"}})
}

type completeUploadRequest struct {
	Title           string `json:"title"`
	Summary         string `json:"summary"`
	Difficulty      int    `json:"difficulty"`
	DurationMinutes int    `json:"duration_minutes"`
}

func (h *Handler) CompleteUpload(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	uploadID := c.Param("upload_id")
	var req completeUploadRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	v, err := h.svc.CompleteUpload(c.Request().Context(), tenantID, userID, uploadID, req.Title, req.Summary, req.Difficulty, req.DurationMinutes)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": v})
}

func (h *Handler) Preview(c echo.Context) error {
	tenantID, _, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	documentID := c.Param("document_id")
	versionNo, err := optionalInt(c.QueryParam("version"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid version"})
	}
	v, reader, mimeType, err := h.svc.Preview(c.Request().Context(), tenantID, documentID, versionNo)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	defer reader.Close()

	res := c.Response()
	res.Header().Set(echo.HeaderContentType, mimeType)
	res.Header().Set(echo.HeaderContentDisposition, "inline; filename=\""+v.FileName+"\"")
	res.WriteHeader(http.StatusOK)
	_, _ = io.Copy(res, reader)
	return nil
}

func (h *Handler) Download(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	documentID := c.Param("document_id")
	versionNo, err := optionalInt(c.QueryParam("version"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid version"})
	}
	v, reader, mimeType, err := h.svc.DownloadWatermarked(c.Request().Context(), tenantID, documentID, userID, versionNo)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	defer reader.Close()

	res := c.Response()
	res.Header().Set(echo.HeaderContentType, mimeType)
	res.Header().Set("X-Watermark", "applied")
	res.Header().Set(echo.HeaderContentDisposition, "attachment; filename=\""+v.FileName+"\"")
	res.WriteHeader(http.StatusOK)
	_, _ = io.Copy(res, reader)
	return nil
}

type shareLinkRequest struct {
	Version *int `json:"version"`
}

func (h *Handler) CreateShareLink(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	documentID := c.Param("document_id")
	var req shareLinkRequest
	_ = c.Bind(&req)
	token, expiresAt, err := h.svc.CreateShareLink(c.Request().Context(), tenantID, userID, documentID, req.Version)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": map[string]any{"token": token, "expires_at": expiresAt}})
}

func (h *Handler) ShareDownload(c echo.Context) error {
	token := c.Param("token")
	tenantID, documentID, versionNo, err := h.svc.ResolveShareDownload(c.Request().Context(), token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "invalid or expired share link"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "share link failed"})
	}
	v, reader, mimeType, err := h.svc.DownloadWatermarked(c.Request().Context(), tenantID, documentID, "share-link", versionNo)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	defer reader.Close()

	res := c.Response()
	res.Header().Set(echo.HeaderContentType, mimeType)
	res.Header().Set("X-Watermark", "applied")
	res.Header().Set(echo.HeaderContentDisposition, "attachment; filename=\""+v.FileName+"\"")
	res.WriteHeader(http.StatusOK)
	_, _ = io.Copy(res, reader)
	return nil
}

func (h *Handler) Search(c echo.Context) error {
	tenantID, _, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	q := c.QueryParam("q")
	if q == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "q is required"})
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	items, err := h.svc.Search(c.Request().Context(), tenantID, q, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "search failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) Versions(c echo.Context) error {
	tenantID, _, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	documentID := c.Param("document_id")
	items, err := h.svc.Versions(c.Request().Context(), tenantID, documentID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": items})
}

type bulkRequest struct {
	DocumentIDs []string `json:"document_ids"`
	CategoryIDs []string `json:"category_ids"`
	TagIDs      []string `json:"tag_ids"`
	Archive     *bool    `json:"archive"`
}

func (h *Handler) Bulk(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req bulkRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if len(req.DocumentIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "document_ids required"})
	}
	if err := h.svc.BulkUpdate(c.Request().Context(), tenantID, userID, req.DocumentIDs, req.CategoryIDs, req.TagIDs, req.Archive); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "updated"}})
}

func (h *Handler) DetectDuplicates(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	n, err := h.svc.DetectDuplicates(c.Request().Context(), tenantID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "duplicate detection failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]int{"flagged": n}})
}

type mergeFlagRequest struct {
	MergeCandidate bool `json:"merge_candidate"`
}

func (h *Handler) SetMergeFlag(c echo.Context) error {
	tenantID, _, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	duplicateID := c.Param("duplicate_id")
	var req mergeFlagRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := h.svc.SetMergeCandidate(c.Request().Context(), tenantID, duplicateID, req.MergeCandidate); err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "duplicate flag not found"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "updated"}})
}

func identity(c echo.Context) (string, string, bool) {
	tenantID, okTenant := c.Get(access.ContextTenantID).(string)
	userID, okUser := c.Get(access.ContextUserID).(string)
	if !okTenant || !okUser || tenantID == "" || userID == "" {
		return "", "", false
	}
	return tenantID, userID, true
}

func optionalInt(raw string) (*int, error) {
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
