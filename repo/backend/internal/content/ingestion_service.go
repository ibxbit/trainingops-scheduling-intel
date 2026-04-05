package content

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var defaultUserAgents = []string{
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
}

func (s *Service) CreateIngestionSource(ctx context.Context, tenantID, userID, name, baseURL string, intervalMinutes, jitterSeconds, rateLimitPerMinute, timeoutSeconds int) (*IngestionSource, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("name is required")
	}
	if err := validatePortalURL(baseURL); err != nil {
		return nil, err
	}
	if intervalMinutes < 5 {
		intervalMinutes = 60
	}
	if jitterSeconds < 0 {
		jitterSeconds = 0
	}
	if rateLimitPerMinute <= 0 {
		rateLimitPerMinute = 6
	}
	if timeoutSeconds < 5 || timeoutSeconds > 120 {
		timeoutSeconds = 20
	}
	return s.repo.CreateIngestionSource(ctx, tenantID, userID, name, baseURL, intervalMinutes, jitterSeconds, rateLimitPerMinute, timeoutSeconds)
}

func (s *Service) ListIngestionSources(ctx context.Context, tenantID string, limit int) ([]IngestionSource, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.ListIngestionSources(ctx, tenantID, limit)
}

func (s *Service) AddIngestionProxy(ctx context.Context, tenantID, proxyURL string) error {
	if err := validateLocalProxyURL(proxyURL); err != nil {
		return err
	}
	return s.repo.AddIngestionProxy(ctx, tenantID, proxyURL)
}

func (s *Service) AddIngestionUserAgent(ctx context.Context, tenantID, userAgent string) error {
	userAgent = strings.TrimSpace(userAgent)
	if userAgent == "" {
		return errors.New("user_agent is required")
	}
	if len(userAgent) > 512 {
		return errors.New("user_agent too long")
	}
	return s.repo.AddIngestionUserAgent(ctx, tenantID, userAgent)
}

func (s *Service) ListIngestionRuns(ctx context.Context, tenantID string, limit int) ([]IngestionRun, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repo.ListIngestionRuns(ctx, tenantID, limit)
}

func (s *Service) SetIngestionManualReview(ctx context.Context, tenantID, sourceID, actorUserID string, approve bool, reason string) error {
	if approve {
		empty := ""
		return s.repo.SetManualReview(ctx, tenantID, sourceID, false, &empty, actorUserID)
	}
	if strings.TrimSpace(reason) == "" {
		reason = "manual review requested"
	}
	return s.repo.SetManualReview(ctx, tenantID, sourceID, true, &reason, actorUserID)
}

func (s *Service) RunDueIngestion(ctx context.Context, tenantID string, maxSources int) ([]IngestionRun, error) {
	if maxSources <= 0 || maxSources > 25 {
		maxSources = 10
	}
	now := time.Now().UTC()
	sources, err := s.repo.DueIngestionSources(ctx, tenantID, now, maxSources)
	if err != nil {
		return nil, err
	}
	out := make([]IngestionRun, 0, len(sources))
	for i := range sources {
		run, err := s.runOne(ctx, tenantID, &sources[i], "scheduled", sources[i].CreatedByUserID)
		if err == nil && run != nil {
			out = append(out, *run)
		}
	}
	return out, nil
}

func (s *Service) RunIngestionNow(ctx context.Context, tenantID, sourceID, actorUserID string) (*IngestionRun, error) {
	source, err := s.repo.IngestionSourceByID(ctx, tenantID, sourceID)
	if err != nil {
		return nil, err
	}
	return s.runOne(ctx, tenantID, source, "manual", actorUserID)
}

func (s *Service) runOne(ctx context.Context, tenantID string, source *IngestionSource, triggerType, actorUserID string) (*IngestionRun, error) {
	if source.PausedForManualReview {
		return nil, errors.New("source paused for manual review")
	}
	now := time.Now().UTC()
	if source.LastRunAt != nil {
		minGap := time.Minute / time.Duration(max(1, source.RateLimitPerMinute))
		if now.Sub(*source.LastRunAt) < minGap {
			next := source.LastRunAt.Add(minGap)
			runID, err := s.repo.StartIngestionRun(ctx, tenantID, source.SourceID, triggerType, nil, nil)
			if err != nil {
				return nil, err
			}
			msg := "rate limited to prevent target overload"
			_ = s.repo.CompleteIngestionRun(ctx, tenantID, runID, "rate_limited", nil, nil, 0, &msg, &next)
			_ = s.repo.AdvanceIngestionSchedule(ctx, tenantID, source.SourceID, next, now)
			runs, _ := s.repo.ListIngestionRuns(ctx, tenantID, 1)
			if len(runs) > 0 {
				return &runs[0], nil
			}
			return nil, nil
		}
	}

	proxyURL, err := s.repo.PickIngestionProxy(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	ua, err := s.repo.PickIngestionUserAgent(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if ua == nil {
		d := defaultUserAgents[randomIndex(len(defaultUserAgents))]
		ua = &d
	}

	runID, err := s.repo.StartIngestionRun(ctx, tenantID, source.SourceID, triggerType, proxyURL, ua)
	if err != nil {
		return nil, err
	}

	nextRun := computeNextRun(source.ScheduleIntervalMinutes, source.ScheduleJitterSeconds)
	client := &http.Client{Timeout: time.Duration(source.RequestTimeoutSeconds) * time.Second}
	if proxyURL != nil {
		pu, err := url.Parse(*proxyURL)
		if err == nil {
			client.Transport = &http.Transport{Proxy: http.ProxyURL(pu)}
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.BaseURL, nil)
	if err != nil {
		msg := err.Error()
		_ = s.repo.CompleteIngestionRun(ctx, tenantID, runID, "failed", nil, nil, 0, &msg, &nextRun)
		_ = s.repo.AdvanceIngestionSchedule(ctx, tenantID, source.SourceID, nextRun, now)
		return nil, err
	}
	if ua != nil {
		req.Header.Set("User-Agent", *ua)
	}
	cookies, _ := s.repo.LoadPortalSessionCookies(ctx, tenantID, source.SourceID)
	if len(cookies) > 0 {
		pairs := make([]string, 0, len(cookies))
		for k, v := range cookies {
			pairs = append(pairs, k+"="+v)
		}
		req.Header.Set("Cookie", strings.Join(pairs, "; "))
	}

	resp, err := client.Do(req)
	if err != nil {
		msg := err.Error()
		_ = s.repo.CompleteIngestionRun(ctx, tenantID, runID, "failed", nil, nil, 0, &msg, &nextRun)
		_ = s.repo.AdvanceIngestionSchedule(ctx, tenantID, source.SourceID, nextRun, now)
		return nil, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if readErr != nil {
		msg := readErr.Error()
		status := resp.StatusCode
		size := int64(len(body))
		_ = s.repo.CompleteIngestionRun(ctx, tenantID, runID, "failed", &status, &size, 0, &msg, &nextRun)
		_ = s.repo.AdvanceIngestionSchedule(ctx, tenantID, source.SourceID, nextRun, now)
		return nil, readErr
	}

	if len(resp.Cookies()) > 0 {
		updated := map[string]string{}
		for k, v := range cookies {
			updated[k] = v
		}
		for _, ck := range resp.Cookies() {
			if ck.Value == "" {
				delete(updated, ck.Name)
				continue
			}
			updated[ck.Name] = ck.Value
		}
		_ = s.repo.SavePortalSessionCookies(ctx, tenantID, source.SourceID, updated, nil)
	}

	statusCode := resp.StatusCode
	responseBytes := int64(len(body))
	if hasBotChallenge(statusCode, body) {
		reason := "bot protection detected (captcha/challenge)"
		_ = s.repo.SetManualReview(ctx, tenantID, source.SourceID, true, &reason, actorUserID)
		_ = s.repo.CompleteIngestionRun(ctx, tenantID, runID, "paused_manual_review", &statusCode, &responseBytes, 0, &reason, nil)
		runs, _ := s.repo.ListIngestionRuns(ctx, tenantID, 1)
		if len(runs) > 0 {
			return &runs[0], nil
		}
		return nil, nil
	}

	if statusCode < 200 || statusCode > 299 {
		msg := "non-success response from partner portal"
		_ = s.repo.CompleteIngestionRun(ctx, tenantID, runID, "failed", &statusCode, &responseBytes, 0, &msg, &nextRun)
		_ = s.repo.AdvanceIngestionSchedule(ctx, tenantID, source.SourceID, nextRun, now)
		return nil, errors.New(msg)
	}

	items := normalizeIngestedRecords(body, resp.Header.Get("Content-Type"), source.SourceID)
	processed := 0
	for _, item := range items {
		path, size, checksum, err := s.storage.SaveIngestedText(tenantID, source.SourceID, item.ExternalID, []byte(item.BodyText))
		if err != nil {
			continue
		}
		item.Checksum = checksum
		if err := s.repo.UpsertIngestedRecord(ctx, tenantID, source.SourceID, actorUserID, path, size, item); err == nil {
			processed++
		}
	}

	_ = s.repo.CompleteIngestionRun(ctx, tenantID, runID, "success", &statusCode, &responseBytes, processed, nil, &nextRun)
	_ = s.repo.AdvanceIngestionSchedule(ctx, tenantID, source.SourceID, nextRun, now)
	runs, _ := s.repo.ListIngestionRuns(ctx, tenantID, 1)
	if len(runs) > 0 {
		return &runs[0], nil
	}
	return nil, nil
}

func computeNextRun(intervalMinutes, jitterSeconds int) time.Time {
	next := time.Now().UTC().Add(time.Duration(intervalMinutes) * time.Minute)
	if jitterSeconds > 0 {
		next = next.Add(time.Duration(randomIndex(jitterSeconds+1)) * time.Second)
	}
	return next
}

func validatePortalURL(v string) error {
	u, err := url.Parse(strings.TrimSpace(v))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return errors.New("invalid base_url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("base_url must use http or https")
	}
	return nil
}

func validateLocalProxyURL(v string) error {
	u, err := url.Parse(strings.TrimSpace(v))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return errors.New("invalid proxy_url")
	}
	host := u.Hostname()
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil {
		if strings.HasSuffix(strings.ToLower(host), ".local") {
			return nil
		}
		return errors.New("proxy_url must be local network address")
	}
	if ip.IsLoopback() || ip.IsPrivate() {
		return nil
	}
	return errors.New("proxy_url must be local network address")
}

func hasBotChallenge(statusCode int, body []byte) bool {
	if statusCode == http.StatusForbidden || statusCode == http.StatusTooManyRequests {
		return true
	}
	l := strings.ToLower(string(body))
	keys := []string{"captcha", "are you human", "bot check", "cloudflare", "security challenge"}
	for _, k := range keys {
		if strings.Contains(l, k) {
			return true
		}
	}
	return false
}

func normalizeIngestedRecords(body []byte, contentType, sourceID string) []IngestedRecord {
	items := make([]IngestedRecord, 0)
	if strings.Contains(strings.ToLower(contentType), "application/json") {
		var raw any
		if err := json.Unmarshal(body, &raw); err == nil {
			records := extractRawItems(raw)
			for _, rec := range records {
				items = append(items, normalizeRawRecord(rec, sourceID))
			}
		}
	}
	if len(items) == 0 {
		text := strings.TrimSpace(string(body))
		if text != "" {
			external := sha256.Sum256(body)
			items = append(items, IngestedRecord{
				ExternalID:   hex.EncodeToString(external[:]),
				Title:        "Partner Content " + time.Now().UTC().Format("2006-01-02"),
				Summary:      trimLen(text, 280),
				Category:     "uncategorized",
				Tags:         []string{"partner"},
				Difficulty:   1,
				DurationMins: 15,
				BodyText:     text,
				Metadata:     map[string]any{"source": sourceID, "normalized": true},
			})
		}
	}
	return items
}

func extractRawItems(raw any) []map[string]any {
	switch v := raw.(type) {
	case map[string]any:
		if items, ok := v["items"].([]any); ok {
			out := make([]map[string]any, 0, len(items))
			for _, item := range items {
				if m, ok := item.(map[string]any); ok {
					out = append(out, m)
				}
			}
			return out
		}
		return []map[string]any{v}
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func normalizeRawRecord(raw map[string]any, sourceID string) IngestedRecord {
	title := pickString(raw, "title", "name", "headline")
	if title == "" {
		title = "Partner Item"
	}
	body := pickString(raw, "content", "body", "text", "description")
	summary := pickString(raw, "summary", "excerpt")
	if summary == "" {
		summary = trimLen(body, 280)
	}
	category := strings.ToLower(strings.TrimSpace(pickString(raw, "category", "type")))
	if category == "" {
		category = "uncategorized"
	}
	tags := normalizeTags(raw["tags"])
	if len(tags) == 0 {
		tags = []string{"partner", category}
	}
	difficulty := clamp(int(pickFloat(raw, "difficulty")), 1, 5)
	duration := clamp(int(pickFloat(raw, "duration_minutes")), 5, 480)
	if duration == 5 && pickFloat(raw, "duration") > 0 {
		duration = clamp(int(pickFloat(raw, "duration")), 5, 480)
	}
	external := pickString(raw, "external_id", "id", "slug", "url")
	if external == "" {
		h := sha256.Sum256([]byte(title + "|" + body))
		external = hex.EncodeToString(h[:])
	}
	metadata := map[string]any{"source": sourceID, "normalized": true}
	for k, v := range raw {
		if _, exists := metadata[k]; !exists {
			metadata[k] = v
		}
	}
	if body == "" {
		body = summary
	}
	return IngestedRecord{
		ExternalID:   external,
		Title:        trimLen(title, 255),
		Summary:      trimLen(summary, 1000),
		Category:     trimLen(category, 120),
		Tags:         tags,
		Difficulty:   difficulty,
		DurationMins: duration,
		BodyText:     trimLen(body, 10000),
		Metadata:     metadata,
	}
}

func pickString(raw map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := raw[k]; ok {
			s := strings.TrimSpace(toString(v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func pickFloat(raw map[string]any, key string) float64 {
	v, ok := raw[key]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		b, _ := json.Marshal(t)
		return string(b)
	default:
		b, _ := json.Marshal(t)
		return string(bytes.Trim(b, `"`))
	}
}

func normalizeTags(v any) []string {
	out := make([]string, 0)
	switch t := v.(type) {
	case []any:
		for _, item := range t {
			tag := strings.ToLower(strings.TrimSpace(toString(item)))
			if tag != "" {
				out = append(out, trimLen(tag, 80))
			}
		}
	case string:
		for _, part := range strings.Split(t, ",") {
			tag := strings.ToLower(strings.TrimSpace(part))
			if tag != "" {
				out = append(out, trimLen(tag, 80))
			}
		}
	}
	return dedupeStrings(out)
}

func dedupeStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func trimLen(v string, n int) string {
	v = strings.TrimSpace(v)
	r := []rune(v)
	if len(r) <= n {
		return v
	}
	return string(r[:n])
}

func clamp(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func randomIndex(max int) int {
	if max <= 1 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
