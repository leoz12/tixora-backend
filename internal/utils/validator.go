package utils

import "regexp"

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail reports whether email is a plausible email address.
func ValidateEmail(email string) bool {
	return emailPattern.MatchString(email)
}

// ValidateName reports whether name has an acceptable length.
func ValidateName(name string) bool {
	return len(name) >= 3 && len(name) <= 255
}

// ValidateQuantity reports whether qty is within the allowed purchase range.
func ValidateQuantity(qty int) bool {
	return qty >= 1 && qty <= 100
}
