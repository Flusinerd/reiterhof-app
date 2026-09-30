package auth

import (
	"bytes"
	"testing"
)

func TestNormalizeLoginCode(t *testing.T) {
	for in, want := range map[string]string{"123456": "123456", "123 456": "123456", " 000 123 ": "000123", "123-456": "123456", "123 456": "123456"} {
		if got, ok := normalizeLoginCode(in); !ok || got != want {
			t.Errorf("%q = %q, %v", in, got, ok)
		}
	}
	for _, in := range []string{"", "12345", "1234567", "12345a", "١٢٣٤٥٦", "12.3456"} {
		if _, ok := normalizeLoginCode(in); ok {
			t.Errorf("%q accepted", in)
		}
	}
}

func TestNewLoginCodeFormat(t *testing.T) {
	seenZero := false
	for i := 0; i < 20000; i++ {
		c, err := newLoginCode()
		if err != nil || len(c) != 6 {
			t.Fatalf("code %q err %v", c, err)
		}
		if _, ok := normalizeLoginCode(c); !ok {
			t.Fatalf("code %q not numeric", c)
		}
		seenZero = seenZero || c[0] == '0'
	}
	if !seenZero {
		t.Fatal("no code with a leading zero in 20000 draws")
	}
	if formatLoginCode("012345") != "012 345" {
		t.Fatal("format")
	}
}

func TestHashLoginCodeBoundToKeyAndEmail(t *testing.T) {
	k := []byte("k")
	h := hashLoginCode(k, "a@example.org", "123456")
	if bytes.Equal(h, hashLoginCode(k, "b@example.org", "123456")) ||
		bytes.Equal(h, hashLoginCode([]byte("other"), "a@example.org", "123456")) ||
		bytes.Equal(h, hashLoginCode(k, "a@example.org", "123457")) {
		t.Fatal("hash does not depend on key, email and code")
	}
}
