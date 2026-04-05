package dashboard

import "testing"

func TestValidFeatureWindow(t *testing.T) {
	for _, v := range []int{7, 30, 90} {
		if !validFeatureWindow(v) {
			t.Fatalf("expected %d to be valid", v)
		}
	}
	for _, v := range []int{0, 1, 14, 365} {
		if validFeatureWindow(v) {
			t.Fatalf("expected %d to be invalid", v)
		}
	}
}
