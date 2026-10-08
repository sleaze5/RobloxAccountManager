package accounts

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

const (
	RoblosecurityCookieName = ".ROBLOSECURITY"
	maxCookieInputBytes     = 16 * 1024
)

var ErrInvalidCookieInput = errors.New("invalid cookie input")

func SplitCookieInputs(input string) []string {
	lines := strings.FieldsFunc(input, func(character rune) bool {
		return character == '\r' || character == '\n'
	})
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func NormalizeCookie(input string) (string, error) {
	if len(input) > maxCookieInputBytes {
		return "", fmt.Errorf("%w: input is too large", ErrInvalidCookieInput)
	}
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("%w: cookie is empty", ErrInvalidCookieInput)
	}
	if strings.ContainsAny(trimmed, "\r\n") {
		return "", fmt.Errorf("%w: paste exactly one cookie without quotes or line breaks", ErrInvalidCookieInput)
	}
	for _, character := range trimmed {
		if character == 0 || unicode.IsControl(character) {
			return "", fmt.Errorf("%w: control characters are not allowed", ErrInvalidCookieInput)
		}
	}
	if len(trimmed) >= 2 && ((trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"') || (trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'')) {
		return "", fmt.Errorf("%w: remove the surrounding quotation marks", ErrInvalidCookieInput)
	}
	if !strings.Contains(trimmed, ";") && !strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(RoblosecurityCookieName)+"=") {
		if err := ValidateCookieValue(trimmed); err != nil {
			return "", err
		}
		return trimmed, nil
	}
	var found string
	count := 0
	for _, part := range strings.Split(trimmed, ";") {
		pair := strings.TrimSpace(part)
		name, value, ok := strings.Cut(pair, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(name), RoblosecurityCookieName) {
			continue
		}
		count++
		found = value
	}
	if count == 0 {
		return "", fmt.Errorf("%w: .ROBLOSECURITY field is missing", ErrInvalidCookieInput)
	}
	if count > 1 {
		return "", fmt.Errorf("%w: duplicate .ROBLOSECURITY fields", ErrInvalidCookieInput)
	}
	if found == "" {
		return "", fmt.Errorf("%w: cookie is empty", ErrInvalidCookieInput)
	}
	if err := ValidateCookieValue(found); err != nil {
		return "", err
	}
	return found, nil
}

func ValidateCookieValue(value string) error {
	if value == "" {
		return fmt.Errorf("%w: cookie is empty", ErrInvalidCookieInput)
	}
	if len(value) > maxCookieInputBytes {
		return fmt.Errorf("%w: input is too large", ErrInvalidCookieInput)
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		// 0x21 to 0x7e is printable ASCII without the space.
		if character < 0x21 || character > 0x7e || character == '"' || character == ',' || character == ';' || character == '\\' {
			return fmt.Errorf("%w: cookie contains a character that Roblox cookies do not use", ErrInvalidCookieInput)
		}
	}
	return nil
}
