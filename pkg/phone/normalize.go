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
	// nonNumericPrefix matches a non-numeric label before the actual number,
	// e.g. "Whats: 62 98576-4545" or "Tel: +55 11 99999-0000"
	nonNumericPrefixPattern = regexp.MustCompile(`^[^0-9+]+`)
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
// It strips non-numeric prefixes (e.g. "Whats: "), WhatsApp JID suffixes,
// formatting characters, and uses the phonenumbers library to parse and validate.
//
// If defaultRegion is non-empty (e.g. "BR"), it is used as a fallback when
// the number has no country-code prefix. The phonenumbers library handles
// region-specific normalization (e.g. Brazilian 8→9 digit migration).
func Normalize(input string, defaultRegion string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("phone: empty input")
	}

	// Strip non-numeric prefix (e.g. "Whats: 62 98576-4545" → "62 98576-4545")
	s := nonNumericPrefixPattern.ReplaceAllString(input, "")

	// Strip WhatsApp JID suffixes
	s = StripJID(s)

	// Strip common formatting characters but preserve leading +
	s = strings.TrimSpace(s)
	hasPlus := strings.HasPrefix(s, "+")

	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
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

	// Parse strategy: try candidates in order, accept the first valid result.
	// 1. If defaultRegion is set, try it first — best for local numbers.
	// 2. Try with no region (works for +E.164 or digits that include country code).
	// 3. If no + and no region, try prepending + as a last-resort hint.
	var attempts []func() (*phonenumbers.PhoneNumber, error)

	if defaultRegion != "" {
		region := defaultRegion
		attempts = append(attempts, func() (*phonenumbers.PhoneNumber, error) {
			return phonenumbers.Parse(candidate, region)
		})
	}

	cand := candidate
	attempts = append(attempts, func() (*phonenumbers.PhoneNumber, error) {
		return phonenumbers.Parse(cand, "")
	})

	if !hasPlus {
		d := digits
		attempts = append(attempts, func() (*phonenumbers.PhoneNumber, error) {
			return phonenumbers.Parse("+"+d, "")
		})
	}

	var num *phonenumbers.PhoneNumber
	var lastErr error
	for _, try := range attempts {
		n, err := try()
		if err == nil && phonenumbers.IsValidNumber(n) {
			num = n
			lastErr = nil
			break
		}
		if err != nil {
			lastErr = err
		}
	}

	if num == nil {
		if lastErr != nil {
			return "", fmt.Errorf("phone: cannot parse %q: %w", input, lastErr)
		}
		return "", fmt.Errorf("phone: invalid number %q", input)
	}

	return phonenumbers.Format(num, phonenumbers.E164), nil
}
