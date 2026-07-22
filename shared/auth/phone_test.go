package auth

import "testing"

func TestCanonicalPhoneDigitsNANP(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"3185125670", "13185125670"},
		{"+13185125670", "13185125670"},
		{"13185125670", "13185125670"},
		{"5551234567", "15551234567"},
		{"+1 3185125670", "13185125670"},
		{"447911123456", "447911123456"},
	}
	for _, tc := range tests {
		if got := CanonicalPhoneDigits(tc.in); got != tc.want {
			t.Fatalf("CanonicalPhoneDigits(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidatePhoneForStorage(t *testing.T) {
	ok, err := ValidatePhoneForStorage("3185125670")
	if err != nil || ok != "13185125670" {
		t.Fatalf("got %q err=%v", ok, err)
	}
	ok, err = ValidatePhoneForStorage("+13185125670")
	if err != nil || ok != "13185125670" {
		t.Fatalf("got %q err=%v", ok, err)
	}
	if _, err := ValidatePhoneForStorage("(318) 512-5670"); err == nil {
		t.Fatal("expected separators to be rejected")
	}
	if _, err := ValidatePhoneForStorage("123"); err == nil {
		t.Fatal("expected short number rejected")
	}
	if _, err := ValidatePhoneForStorage("0123456789"); err == nil {
		t.Fatal("expected invalid NANP area-code lead rejected")
	}
}

func TestPhonesMatchEquatesNANPForms(t *testing.T) {
	if !PhonesMatch("3185125670", "13185125670") {
		t.Fatal("10-digit and 11-digit NANP must match")
	}
	if !PhonesMatch("+1 318-512-5670", "3185125670") {
		t.Fatal("formatted and bare NANP must match after digit strip")
	}
	if PhonesMatch("3185125670", "3185125671") {
		t.Fatal("different numbers must not match")
	}
	if PhonesMatch("", "3185125670") {
		t.Fatal("empty must not match")
	}
}
