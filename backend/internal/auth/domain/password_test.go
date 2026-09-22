package domain

import (
	"strings"
	"testing"
)

func TestPasswordPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, password string
		valid          bool
	}{
		{"valid", "Kuat-sekali-2026!", true},
		{"too short", "Pendek1!", false},
		{"missing upper", "semuakecil-2026!", false},
		{"missing number", "TanpaAngka!!", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePassword(tc.password)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
}

func TestArgon2idHashesAndVerifiesWithoutPlaintext(t *testing.T) {
	h := NewPasswordHasher()
	encoded, err := h.Hash("Kuat-sekali-2026!")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(encoded, "Kuat-sekali-2026!") || !strings.HasPrefix(encoded, "$argon2id$") {
		t.Fatalf("unsafe or invalid hash: %q", encoded)
	}
	if ok, err := h.Verify("Kuat-sekali-2026!", encoded); err != nil || !ok {
		t.Fatalf("verification failed: ok=%v err=%v", ok, err)
	}
	if ok, _ := h.Verify("Salah-sekali-2026!", encoded); ok {
		t.Fatal("wrong password accepted")
	}
}
