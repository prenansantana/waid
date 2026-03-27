// Package phone provides phone number normalization and validation utilities.
package phone

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

var (
	jidSuffixes = []string{"@s.whatsapp.net", "@c.us", "@g.us"}
	e164Pattern = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)
)

// StripJID removes WhatsApp JID suffixes from the input string.
func StripJID(input string) string {
	for _, suffix := range jidSuffixes {
		if strings.HasSuffix(input, suffix) {
			return strings.TrimSuffix(input, suffix)
		}
	}
	return input
}

// IsE164 returns true if phone is a valid E.164 formatted number.
func IsE164(phone string) bool {
	return e164Pattern.MatchString(phone)
}

// Normalize converts any phone number format to E.164.
// It strips WhatsApp JID suffixes, strips formatting characters, and uses
// the phonenumbers library to parse and validate the result.
func Normalize(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("phone: empty input")
	}

	// Strip WhatsApp JID suffixes
	s := StripJID(input)

	// Strip common formatting characters but preserve leading +
	s = strings.TrimSpace(s)
	hasPlus := strings.HasPrefix(s, "+")

	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if digits == "" {
		return "", fmt.Errorf("phone: no digits found in %q", input)
	}

	// Re-attach the + for the parser
	candidate := digits
	if hasPlus {
		candidate = "+" + digits
	}

	// Try to parse with no default region (requires country code prefix)
	num, err := phonenumbers.Parse(candidate, "")
	if err != nil {
		// If it failed and there was no +, try prepending + as a hint
		if !hasPlus {
			num, err = phonenumbers.Parse("+"+digits, "")
		}
		if err != nil {
			return "", fmt.Errorf("phone: cannot parse %q: %w", input, err)
		}
	}

	if !phonenumbers.IsValidNumber(num) {
		return "", fmt.Errorf("phone: invalid number %q", input)
	}

	return phonenumbers.Format(num, phonenumbers.E164), nil
}
