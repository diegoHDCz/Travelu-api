package user

import "regexp"

// phoneRegex matches E.164 numbers: an optional leading '+' followed by 8 to
// 15 digits, the first of which is non-zero.
var phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{7,14}$`)

// SanitizePhone strips everything but digits from raw, keeping a leading
// '+' when present, so formatted input such as "+55 (11) 99999-8888" is
// normalized to "+5511999998888" before validation and storage.
func SanitizePhone(raw string) string {
	out := make([]byte, 0, len(raw))
	for i, r := range raw {
		switch {
		case r == '+' && i == 0:
			out = append(out, '+')
		case r >= '0' && r <= '9':
			out = append(out, byte(r))
		}
	}
	return string(out)
}

// ValidPhone reports whether phone — already sanitized by SanitizePhone —
// is a valid E.164 number.
func ValidPhone(phone string) bool {
	return phoneRegex.MatchString(phone)
}
