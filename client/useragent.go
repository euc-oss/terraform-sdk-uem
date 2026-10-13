package client

import (
	"strings"
	"unicode"
)

// userAgentHeader is the request header every SDK request carries.
const userAgentHeader = "User-Agent"

// DefaultUserAgent returns the SDK's own User-Agent product token,
// "uem-sdk-go/<version>". Every request the SDK sends carries it, API calls and
// the OAuth2 token call alike.
func DefaultUserAgent() string {
	return "uem-sdk-go/" + sdkVersion()
}

// SanitizeUserAgentPrefix cleans a caller-supplied product token (or several,
// space separated) for use in a User-Agent header:
//
//   - leading and trailing white space is dropped and every run of white space
//     collapses to one space;
//   - control characters (CR, LF, NUL, DEL and the rest), which could split or
//     corrupt the header, are treated as white space, so "a\r\nb" becomes "a b";
//   - other characters outside ASCII are dropped.
//
// The result is at most MaxUserAgentPrefixLen characters (cut at a token boundary
// when it can be) and empty when nothing usable remains. It never contains two
// consecutive spaces, a leading or a trailing space.
func SanitizeUserAgentPrefix(s string) string {
	var b strings.Builder
	pendingSpace := false
	for _, r := range s {
		switch {
		case r == ' ' || unicode.IsSpace(r) || unicode.IsControl(r):
			pendingSpace = b.Len() > 0
		case r > unicode.MaxASCII:
			// Not representable in a header value; drop it without splitting
			// the token around it.
		default:
			if pendingSpace {
				b.WriteByte(' ')
				pendingSpace = false
			}
			b.WriteRune(r)
		}
	}
	return capUserAgentPrefix(b.String())
}

// MaxUserAgentPrefixLen bounds the caller's part of the User-Agent.
const MaxUserAgentPrefixLen = 256

// capUserAgentPrefix limits s (already sanitized, ASCII) to MaxUserAgentPrefixLen
// characters, cutting at the last space before the limit when there is one so
// that no token is left half written.
func capUserAgentPrefix(s string) string {
	if len(s) <= MaxUserAgentPrefixLen {
		return s
	}
	cut := s[:MaxUserAgentPrefixLen]
	if next := s[MaxUserAgentPrefixLen]; next != ' ' {
		if i := strings.LastIndexByte(cut, ' '); i > 0 {
			cut = cut[:i]
		}
	}
	return strings.TrimRight(cut, " ")
}

// BuildUserAgent returns the User-Agent value for a client: the caller's product
// token(s) after sanitizing, followed by the SDK's own token, for example
// "my-app/1.2.0 my-framework/3.4.0 uem-sdk-go/26.2.0".
// When the caller's token sanitizes to nothing, only the SDK token is returned.
// The SDK never adds a caller prefix of its own: whoever embeds the SDK passes it.
func BuildUserAgent(callerTokens string) string {
	prefix := SanitizeUserAgentPrefix(callerTokens)
	if prefix == "" {
		return DefaultUserAgent()
	}
	return prefix + " " + DefaultUserAgent()
}
