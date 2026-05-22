package validation

import "testing"

func TestEmail(t *testing.T) {
	if !Email("user@example.com") {
		t.Fatal("valid email rejected")
	}
	if Email("bad") {
		t.Fatal("invalid email accepted")
	}
}

func TestPassword(t *testing.T) {
	if !Password("123456") {
		t.Fatal("6 char password should pass")
	}
	if Password("12345") {
		t.Fatal("short password should fail")
	}
}

func TestPositiveInt(t *testing.T) {
	if !PositiveInt(1) || PositiveInt(0) || PositiveInt(-1) {
		t.Fatal("positive int validation wrong")
	}
}
