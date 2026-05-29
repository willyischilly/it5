package validation

import (
	"net/mail"
	"strings"
)

func Email(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	_, err := mail.ParseAddress(email)
	return err == nil
}

func Password(password string) bool {
	return len(password) >= 6
}

func NonEmpty(s string) bool {
	return strings.TrimSpace(s) != ""
}

func PositiveInt(n int) bool {
	return n >= 1
}

func MaxLen(s string, max int) bool {
	s = strings.TrimSpace(s)
	return s != "" && len(s) <= max
}

// PersonName — фамилия, имя или отчество (до 100 символов).
func PersonName(s string) bool {
	return MaxLen(s, 100)
}
