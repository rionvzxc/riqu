package riqu

import (
	"io"
	"sync"
)

// QResponse contains the metadata and body of an HTTP response.
type QResponse struct {
	statusCode    int
	status        string
	resHeader     THeader
	resCookie     []*TCookie
	resUrl        string
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
// Url returns the URL used for this response.
func (this *QResponse) Url() string {
	if this == nil {
		return ""
	}
	return this.resUrl
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
