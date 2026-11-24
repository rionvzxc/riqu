package riqu

import (
	"io"
	"sync"
)

// QResponse contains the metadata and body of an HTTP response.
type QResponse struct {
	statusCode    int
	status        string
	contentLength int64
	resHeader     THeader
	resCookie     TCookies
	resURL        string
	resMethod     string
	resQuery      TValues
	queryOnce     sync.Once
	queryErr      error
	bodyRead      bool
	body          io.ReadCloser
}
// StatusCode returns the numeric HTTP status code.
func (this *QResponse) StatusCode() int {
	if this == nil {
		return 0
	}
	return this.statusCode
}
// Status returns the HTTP response status, including its numeric code.
func (this *QResponse) Status() string {
	if this == nil {
		return ""
	}
	return this.status
}
// ContentLength returns the response body length in bytes, or -1 when unknown.
// A nil response also returns -1.
func (this *QResponse) ContentLength() int64 {
	if this == nil {
		return -1
	}
	return this.contentLength
}
// URL returns the URL used for this response.
func (this *QResponse) URL() string {
	if this == nil {
		return ""
	}
	return this.resURL
}
// Method returns the HTTP method used for this response.
func (this *QResponse) Method() string {
	if this == nil {
		return ""
	}
	return this.resMethod
}
func (this *QResponse) close() error {
	if this == nil || this.body == nil {
		return nil
	}
	this.bodyRead = true
	return this.body.Close()
}
