package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tixora/internal/utils"
)

func TestValidateEmail(t *testing.T) {
	cases := []struct {
		email string
		want  bool
	}{
		{"user@example.com", true},
		{"first.last+tag@sub.example.co.id", true},
		{"invalid", false},
		{"missing-at.example.com", false},
		{"missing-domain@", false},
		{"", false},
		{"user@example", false},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, utils.ValidateEmail(tc.email), "email %q", tc.email)
	}
}

func TestValidateName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"Bob", true},
		{"Al", false}, // too short (2 chars)
		{"", false},   // empty
		{string(make([]byte, 255)), true},
		{string(make([]byte, 256)), false},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, utils.ValidateName(tc.name), "name length %d", len(tc.name))
	}
}

func TestValidateQuantity(t *testing.T) {
	cases := []struct {
		qty  int
		want bool
	}{
		{0, false},
		{1, true},
		{100, true},
		{101, false},
		{-1, false},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, utils.ValidateQuantity(tc.qty), "qty %d", tc.qty)
	}
}
