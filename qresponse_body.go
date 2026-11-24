package riqu

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Data reads and closes the response body.
func (this *QResponse) Data() ([]byte, error) {
	var data []byte
	err := this.consumeBody(func(reader io.Reader) error {
		var err error
		data, err = io.ReadAll(reader)
		return err
	})
	return data, err
}
// WriteTo copies the response body to writer and closes the body.
// It returns the number of bytes written and does not close writer.
func (this *QResponse) WriteTo(writer io.Writer) (written int64, err error) {
	if this == nil {
		return 0, fmt.Errorf("riqu: response is nil")
	}
	if this.isBodyRead {
		return 0, fmt.Errorf("riqu: response body already read")
	}
	if this.body == nil {
		return 0, fmt.Errorf("riqu: response body unavailable")
	}
	if writer == nil {
		return 0, fmt.Errorf("riqu: writer is nil")
	}
	err = this.consumeBody(func(reader io.Reader) error {
		var copyErr error
		written, copyErr = io.Copy(writer, reader)
		return copyErr
	})
	return written, err
}
// Text reads the response body as a string.
func (this *QResponse) Text() (string, error) {
	rawData, err := this.Data()

	if err != nil {
		return "", err
	}
	return string(rawData), nil
}
// BindJSON decodes the response body into target.
func (this *QResponse) BindJSON(target any) (err error) {
	return this.consumeBody(func(reader io.Reader) error {
		decoder := json.NewDecoder(reader)
		decodeErr := decoder.Decode(target)

		if decodeErr == nil {
			_, decodeErr = decoder.Token()

			if decodeErr == io.EOF {
				decodeErr = nil
			} else if decodeErr == nil {
				decodeErr = errors.New("riqu: response body contains additional JSON data")
			}
		}
		return decodeErr
	})
}
