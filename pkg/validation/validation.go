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
