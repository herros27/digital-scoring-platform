package auth

import (
	"strings"
	"testing"
)

func TestPINLengthByRole(t *testing.T) {
	for role, want := range map[string]int{"judge": 6, "juri": 6, "operator": 6, "wasit": 8, "referee": 8} {
		if got := PINLength(role); got != want {
			t.Errorf("PINLength(%q) = %d, want %d", role, got, want)
		}
	}
}

func TestValidatePIN(t *testing.T) {
	cases := []struct {
		role, pin string
		ok        bool
	}{
		{"judge", "123456", true},
		{"judge", "12345", false},
		{"judge", "1234567", false},
		{"judge", "12345a", false},
		{"wasit", "12345678", true},
		{"wasit", "123456", false}, // wasit wajib 8 digit
		{"wasit", "1234567a", false},
		{"judge", "", false},
	}
	for _, c := range cases {
		if err := ValidatePIN(c.role, c.pin); (err == nil) != c.ok {
			t.Errorf("ValidatePIN(%q, %q) err=%v, ok diharapkan %v", c.role, c.pin, err, c.ok)
		}
	}
}

func TestGeneratePIN(t *testing.T) {
	for _, digits := range []int{6, 8} {
		seen := map[string]bool{}
		for i := 0; i < 200; i++ {
			pin, err := GeneratePIN(digits)
			if err != nil {
				t.Fatal(err)
			}
			if len(pin) != digits || strings.Trim(pin, "0123456789") != "" {
				t.Fatalf("PIN %q bukan %d digit angka", pin, digits)
			}
			seen[pin] = true
		}
		if len(seen) < 150 {
			t.Errorf("PIN %d digit kurang acak: hanya %d unik dari 200", digits, len(seen))
		}
	}
}

func TestPinLookup(t *testing.T) {
	a := PinLookup("123456")
	if len(a) != 64 {
		t.Fatalf("panjang lookup = %d, want 64 (hex SHA-256)", len(a))
	}
	if a != PinLookup("123456") {
		t.Error("lookup harus deterministik")
	}
	if a == PinLookup("123457") {
		t.Error("PIN berbeda harus menghasilkan lookup berbeda")
	}
	if strings.Contains(a, "123456") {
		t.Error("lookup tidak boleh memuat PIN mentah")
	}

	old := jwtSecret
	defer func() { jwtSecret = old }()
	SetJWTSecret("secret-lain")
	if PinLookup("123456") == a {
		t.Error("lookup harus bergantung pada secret server")
	}
}
