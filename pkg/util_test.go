package utils

import (
	"testing"
)

func TestFindText(t *testing.T) {
	println(FindTextPS("https://pkg.go.dev/strings#TrimPrefix", "/", "#"))
	println(FindTextP("https://pkg.go.dev/strings#TrimPrefix", "/"))
}
