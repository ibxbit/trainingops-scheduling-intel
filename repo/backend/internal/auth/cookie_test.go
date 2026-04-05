package auth

import "testing"

func TestCookieRoundTrip(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"
	raw := "token-value"
	v := CookieValue(tenantID, raw)
	gotTenant, gotRaw := ParseCookieValue(v)
	if gotTenant != tenantID || gotRaw != raw {
		t.Fatalf("unexpected parse result: tenant=%s token=%s", gotTenant, gotRaw)
	}
}

func TestParseCookieInvalid(t *testing.T) {
	tests := []string{"", "v1", "v1.", "v1.onlytenant", "v1..token", "invalid"}
	for _, in := range tests {
		ten, tok := ParseCookieValue(in)
		if ten != "" || tok != "" {
			t.Fatalf("expected empty parse for %q", in)
		}
	}
}
