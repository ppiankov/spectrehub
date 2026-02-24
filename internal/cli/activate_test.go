package cli

import (
	"testing"
)

func TestKeyPatternValid(t *testing.T) {
	valid := []string{
		"sh_live_0123456789abcdef0123456789abcdef",
		"sh_test_0123456789abcdef0123456789abcdef",
		"sh_live_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"sh_test_00000000000000000000000000000000",
	}
	for _, k := range valid {
		if !keyPattern.MatchString(k) {
			t.Errorf("expected valid key: %s", k)
		}
	}
}

func TestKeyPatternInvalid(t *testing.T) {
	invalid := []string{
		"",
		"sh_live_short",
		"sh_live_0123456789ABCDEF0123456789abcdef",  // uppercase
		"xx_live_0123456789abcdef0123456789abcdef",  // wrong prefix
		"sh_prod_0123456789abcdef0123456789abcdef",  // wrong env
		"sh_live_0123456789abcdef0123456789abcde",   // 31 chars
		"sh_live_0123456789abcdef0123456789abcdefg", // 33 chars
		"not_a_key",
	}
	for _, k := range invalid {
		if keyPattern.MatchString(k) {
			t.Errorf("expected invalid key: %s", k)
		}
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"sh_live_0123456789abcdef0123456789abcdef", "sh_live_01...cdef"},
		{"sh_test_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sh_test_aa...aaaa"},
		{"short", "****"},
	}
	for _, tt := range tests {
		got := maskKey(tt.input)
		if got != tt.want {
			t.Errorf("maskKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
