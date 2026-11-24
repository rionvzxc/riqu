package riqu

import "io"

type tRequestBodyType uint

const (
	vRequestBodyData tRequestBodyType = iota
	vRequestBodyText
	vRequestBodyHTML
	vRequestBodyJSON
	vRequestBodyForm
	vRequestBodyMultipart
	vRequestBodyReader
)
// tRequestBody stores request body data and its content type.
type tRequestBody struct {
	// bodyType selects how value is encoded when sent.
	bodyType tRequestBodyType
	// value contains raw bytes or the value to encode.
	value any
	// contentType is the media type sent with the request body.
	contentType string
}
type tPreparedBody struct {
	data        []byte
	contentType string
	multipart   *TMultipart
	reader      *tRequestReader
}
type tRequestReader struct {
	reader io.Reader
	isUsed   bool
}
