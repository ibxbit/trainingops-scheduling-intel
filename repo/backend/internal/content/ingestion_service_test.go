package content

import (
	"net/http"
	"testing"
)

func TestValidateLocalProxyURL(t *testing.T) {
	valid := []string{
		"http://localhost:8080",
		"http://127.0.0.1:8080",
		"http://192.168.1.10:3128",
		"http://myproxy.local:8080",
	}
	for _, v := range valid {
		if err := validateLocalProxyURL(v); err != nil {
			t.Fatalf("expected valid proxy %s: %v", v, err)
		}
	}

	invalid := []string{"http://8.8.8.8:8080", "https://example.com:8080"}
	for _, v := range invalid {
		if err := validateLocalProxyURL(v); err == nil {
			t.Fatalf("expected invalid proxy %s", v)
		}
	}
}

func TestHasBotChallenge(t *testing.T) {
	if !hasBotChallenge(http.StatusForbidden, []byte("ok")) {
		t.Fatal("expected forbidden status to be challenge")
	}
	if !hasBotChallenge(http.StatusOK, []byte("Please solve CAPTCHA")) {
		t.Fatal("expected captcha text to be challenge")
	}
	if hasBotChallenge(http.StatusOK, []byte("normal payload")) {
		t.Fatal("expected normal payload not to be challenge")
	}
}
