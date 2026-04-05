package security

import (
	"errors"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const MinPasswordLength = 12

func ValidatePasswordRules(password string) error {
	if utf8.RuneCountInString(password) < MinPasswordLength {
		return errors.New("password must be at least 12 characters")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	if err := ValidatePasswordRules(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
