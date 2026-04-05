package content

import "testing"

func TestNormalizeIngestedRecordsFallbackToPlainText(t *testing.T) {
	body := []byte("plain text payload from partner")
	items := normalizeIngestedRecords(body, "text/plain", "source-1")
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].BodyText == "" {
		t.Fatal("expected fallback body text to be populated")
	}
	if items[0].Title == "" {
		t.Fatal("expected fallback title")
	}
}

func TestNormalizeIngestedRecordsInvalidJSONStillFallsBack(t *testing.T) {
	body := []byte("{invalid-json")
	items := normalizeIngestedRecords(body, "application/json", "source-1")
	if len(items) != 1 {
		t.Fatalf("expected 1 fallback item, got %d", len(items))
	}
}

func TestNormalizeIngestedRecordsJSONArray(t *testing.T) {
	body := []byte(`[
  {"id":"a1","title":"Intro","content":"Body","category":"ops","tags":["ops","onboarding"],"difficulty":3,"duration_minutes":45}
]`)
	items := normalizeIngestedRecords(body, "application/json", "source-1")
	if len(items) != 1 {
		t.Fatalf("expected 1 record, got %d", len(items))
	}
	if items[0].ExternalID != "a1" {
		t.Fatalf("expected external id a1, got %s", items[0].ExternalID)
	}
	if items[0].Difficulty != 3 {
		t.Fatalf("expected difficulty 3, got %d", items[0].Difficulty)
	}
}

func TestHasBotChallengeTextSignals(t *testing.T) {
	if !hasBotChallenge(200, []byte("Please solve CAPTCHA to continue")) {
		t.Fatal("expected captcha text to trigger bot challenge")
	}
	if hasBotChallenge(200, []byte("normal content")) {
		t.Fatal("did not expect normal content to trigger bot challenge")
	}
}
