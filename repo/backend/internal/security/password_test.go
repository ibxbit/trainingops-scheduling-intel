package security

import "testing"

func TestValidatePasswordRules(t *testing.T) {
	if err := ValidatePasswordRules("short"); err == nil {
		t.Fatal("expected short password to fail")
	}
	if err := ValidatePasswordRules("long-enough-password"); err != nil {
		t.Fatalf("expected valid password, got error: %v", err)
	}
}

func TestHashAndComparePassword(t *testing.T) {
	h, err := HashPassword("StrongPass123")
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}
	if err := ComparePassword(h, "StrongPass123"); err != nil {
		t.Fatalf("compare failed: %v", err)
	}
	if err := ComparePassword(h, "WrongPass123"); err == nil {
		t.Fatal("expected mismatch password to fail")
	}
}
