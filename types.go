package riqu

import (
	"net/http"
	"net/http/cookiejar"
)

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

// TRedirectResponseMode controls which redirect responses are delivered to callbacks.
type TRedirectResponseMode uint

// TRedirect configures how requests follow redirects.
type TRedirect struct {
	// Max is the maximum number of redirects to follow.
	Max int
	// ResponseMode selects which redirect responses are delivered to callbacks.
	ResponseMode TRedirectResponseMode
}
// tRequestBody stores request body data and its content type.
type tRequestBody struct {
	// bodyType selects how value is encoded when sent.
	bodyType tRequestBodyType
	// value contains raw bytes or the value to encode.
	value any
	// contentType is the media type sent with the request body.
	contentType string
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
