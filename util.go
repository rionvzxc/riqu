package riqu

// IsRedirectStatus reports whether statusCode is an HTTP redirect status.
func IsRedirectStatus(statusCode int) bool {
	switch statusCode {
	case 301, 302, 303, 307, 308:
		return true
	}
	return false
}
