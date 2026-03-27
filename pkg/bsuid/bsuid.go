// Package bsuid provides parsing and validation of Meta Business-Scoped User IDs.
//
// A BSUID has the format: {CountryCode}.{AlphanumericID}
// Example: BR.abc123def456, US.xyz789ghi012
package bsuid

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	maxBSUIDLength = 64
	minIDLength    = 8
)

// BSUID represents a parsed Meta Business-Scoped User ID.
type BSUID struct {
	CountryCode string // ISO 3166-1 alpha-2 (e.g., BR, US)
	ID          string // alphanumeric string
	Raw         string // original full string
}

// Parse parses and validates a BSUID string, returning the structured result or an error.
func Parse(input string) (*BSUID, error) {
	if input == "" {
		return nil, fmt.Errorf("bsuid: empty input")
	}
	if len(input) > maxBSUIDLength {
		return nil, fmt.Errorf("bsuid: input too long (max %d chars)", maxBSUIDLength)
	}

	parts := strings.SplitN(input, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("bsuid: missing dot separator in %q", input)
	}

	cc := parts[0]
	id := parts[1]

	if len(cc) != 2 {
		return nil, fmt.Errorf("bsuid: country code must be exactly 2 characters, got %q", cc)
	}
	for _, r := range cc {
		if r < 'A' || r > 'Z' {
			return nil, fmt.Errorf("bsuid: country code must be uppercase ASCII letters, got %q", cc)
		}
	}

	if len(id) < minIDLength {
		return nil, fmt.Errorf("bsuid: ID part too short (min %d chars), got %q", minIDLength, id)
	}
	for _, r := range id {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return nil, fmt.Errorf("bsuid: ID part must be alphanumeric, got %q", id)
		}
	}

	return &BSUID{
		CountryCode: cc,
		ID:          id,
		Raw:         input,
	}, nil
}

// IsValid returns true if input is a syntactically valid BSUID.
func IsValid(input string) bool {
	_, err := Parse(input)
	return err == nil
}

// IsBSUID returns true if input looks like a BSUID rather than a phone number.
// It checks for the characteristic {2-letter-code}.{alphanumeric} pattern.
func IsBSUID(input string) bool {
	if len(input) < 2+1+minIDLength {
		return false
	}
	dotIdx := strings.Index(input, ".")
	if dotIdx != 2 {
		return false
	}
	return IsValid(input)
}
