package gitsubstrate

import "testing"

func TestSanitizeID(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"feature/login-bug", "feature_login-bug"},
		{"ABC-123", "ABC-123"},
		{"main", "main"},
		{"", "_"},
		{"!@#$%^&*()", "__________"},
		{"with.dots.and-dashes_and_underscores", "with.dots.and-dashes_and_underscores"},
		{"branch with spaces", "branch_with_spaces"},
	}
	for _, c := range cases {
		if got := SanitizeID(c.in); got != c.want {
			t.Errorf("SanitizeID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSanitizeID_Idempotent(t *testing.T) {
	in := "feature/some-branch_name.v2"
	first := SanitizeID(in)
	second := SanitizeID(first)
	if first != second {
		t.Errorf("SanitizeID not idempotent: first=%q second=%q", first, second)
	}
}
