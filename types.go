package riqu

import (
	"net/http"
	"net/http/cookiejar"
)

// TTransferInfo describes one request or response body transfer.
type TTransferInfo struct {
	// URL is the destination of this transfer, including a redirect destination.
	URL string
	// Method is the HTTP method used for this transfer.
	Method string
	// Transferred counts body bytes read, not bytes committed by a destination writer.
	Transferred int64
	// Total is -1 when the body length is unknown.
	Total int64
	// Err is populated for an unsuccessful End event.
	Err error
}

// TMap is a generic map with string keys.
type TMap = map[string]any

// TJarOptions configures a cookie jar.
type TJarOptions = cookiejar.Options

// TTransportOptions configures a reusable HTTP transport.
type TTransportOptions struct {
	// ProxyURL overrides the environment proxy when supplied.
	ProxyURL string
	// CAFile adds PEM-encoded CA certificates to the system trust pool.
	CAFile string
}

// TCookie represents an HTTP cookie.
type TCookie = http.Cookie

// TRedirectOptions configures how requests follow redirects.
type TRedirectOptions struct {
	// Max is the maximum number of redirects to follow.
	Max int
	// Responses selects which redirect responses are delivered to callbacks.
	Responses TResponseMode
}
// TFileOptions configures how a response body is saved to a file.
type TFileOptions struct {
	// Dir is the directory where the response file is saved.
	Dir string
	// Filename overrides the filename inferred from the response.
	Filename string
	// DefaultFilename is used when the response does not provide a filename.
	DefaultFilename string
}
// TMultipart stores multipart form fields and file paths.
type TMultipart struct {
	// Form contains regular multipart form fields.
	Form TValues
	// File maps multipart field names to file paths.
	File TValues
}
