package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tixora/internal/utils"
)

func TestSlugify(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Music & Concerts", "music-concerts"},
		{"  Leading and trailing  ", "leading-and-trailing"},
		{"Already-Slugged", "already-slugged"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"Special!!!Characters???", "special-characters"},
		{"UPPERCASE", "uppercase"},
		{"123 Numbers OK", "123-numbers-ok"},
		{"---trim-hyphens---", "trim-hyphens"},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, utils.Slugify(tc.in), "input %q", tc.in)
	}
}
