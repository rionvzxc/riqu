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
	isBodyRead      bool
	body          io.ReadCloser
	transferBody      *tTransferBody
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
	this.isBodyRead = true
	return this.body.Close()
}
