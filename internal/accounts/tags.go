package accounts

import (
	"fmt"
	"regexp"
)

const maxTagNameLength = 40

var validTagNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func ValidateTagName(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("%w: tag name is empty", ErrInvalidTagName)
	}
	if len(input) > maxTagNameLength {
		return "", fmt.Errorf("%w: tag name is longer than %d characters", ErrInvalidTagName, maxTagNameLength)
	}
	if !validTagNamePattern.MatchString(input) {
		return "", fmt.Errorf("%w: use lowercase letters and numbers, with single hyphens between words", ErrInvalidTagName)
	}
	switch input {
	case "all", "favorite", "favorites":
		return "", fmt.Errorf("%w: that name is reserved for a built-in tag", ErrInvalidTagName)
	}
	return input, nil
}
