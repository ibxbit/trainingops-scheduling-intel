package observability

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"trainingops/backend/internal/dashboard"
)

func writeCSV(path string, summary dashboard.DailySummary, kpis []dashboard.KPI, heatmap []dashboard.HeatmapCell) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	_ = w.Write([]string{"section", "key", "value", "numerator", "denominator"})
	_ = w.Write([]string{"summary", "metric_date", summary.MetricDate, "", ""})
	_ = w.Write([]string{"summary", "todays_sessions", itoa(summary.TodaysSessions), "", ""})
	_ = w.Write([]string{"summary", "pending_approvals", itoa(summary.PendingApprovals), "", ""})
	for _, k := range kpis {
		_ = w.Write([]string{"kpi", k.MetricKey, fmt.Sprintf("%.6f", k.MetricValue), fmt.Sprintf("%.6f", k.Numerator), fmt.Sprintf("%.6f", k.Denominator)})
	}
	_ = w.Write([]string{"heatmap", "hour_bucket", "room_id", "sessions", "occupancy"})
	for _, h := range heatmap {
		room := ""
		if h.RoomID != nil {
			room = *h.RoomID
		}
		_ = w.Write([]string{"heatmap", itoa(h.HourBucket), room, itoa(h.SessionsCount), fmt.Sprintf("%.6f", h.OccupancyRate)})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return 0, err
	}
	st, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return st.Size(), nil
}

func writePDF(path string, summary dashboard.DailySummary, kpis []dashboard.KPI, heatmap []dashboard.HeatmapCell) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}
	lines := []string{
		"TrainingOps Dashboard Report",
		"Date: " + summary.MetricDate,
		"Today's Sessions: " + itoa(summary.TodaysSessions),
		"Pending Approvals: " + itoa(summary.PendingApprovals),
		"",
		"KPI Tiles:",
	}
	for _, k := range kpis {
		lines = append(lines, fmt.Sprintf("- %s = %.4f", k.MetricKey, k.MetricValue))
	}
	lines = append(lines, "", "Occupancy Heatmap:")
	for _, h := range heatmap {
		room := "all"
		if h.RoomID != nil {
			room = *h.RoomID
		}
		lines = append(lines, fmt.Sprintf("- hour %02d room %s sessions %d occ %.2f", h.HourBucket, room, h.SessionsCount, h.OccupancyRate))
	}

	content := buildPDFText(lines)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return 0, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return st.Size(), nil
}

func buildPDFText(lines []string) []byte {
	var text strings.Builder
	text.WriteString("BT\n/F1 11 Tf\n50 780 Td\n")
	first := true
	for _, l := range lines {
		esc := strings.ReplaceAll(strings.ReplaceAll(l, "\\", "\\\\"), "(", "\\(")
		esc = strings.ReplaceAll(esc, ")", "\\)")
		if !first {
			text.WriteString("0 -14 Td\n")
		}
		text.WriteString("(" + esc + ") Tj\n")
		first = false
	}
	text.WriteString("ET")
	stream := text.String()

	objs := []string{
		"1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n",
		"2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n",
		"3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >> endobj\n",
		fmt.Sprintf("4 0 obj << /Length %d >> stream\n%s\nendstream endobj\n", len(stream), stream),
		"5 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj\n",
	}

	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs)+1)
	for i, obj := range objs {
		offsets[i+1] = b.Len()
		b.WriteString(obj)
	}
	xref := b.Len()
	b.WriteString("xref\n0 6\n")
	b.WriteString("0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	b.WriteString("trailer << /Size 6 /Root 1 0 R >>\n")
	b.WriteString(fmt.Sprintf("startxref\n%d\n%%%%EOF\n", xref))
	return b.Bytes()
}
