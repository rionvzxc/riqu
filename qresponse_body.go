package riqu

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Data reads and closes the response body.
func (this *QResponse) Data() ([]byte, error) {
	if this == nil {
		return nil, fmt.Errorf("riqu: response is nil")
	}
	if this.bodyRead {
		return nil, fmt.Errorf("riqu: response body already read")
	}
	if this.body == nil {
		return nil, fmt.Errorf("riqu: response body unavailable")
	}
	this.bodyRead = true
	data, readErr := io.ReadAll(this.body)
	closeErr := this.body.Close()
	return data, errors.Join(readErr, closeErr)
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
func (this *QResponse) BindJSON(target any) error {
	if this == nil {
		return fmt.Errorf("riqu: response is nil")
	}
	if this.bodyRead {
		return fmt.Errorf("riqu: response body already read")
	}
	if this.body == nil {
		return fmt.Errorf("riqu: response body unavailable")
	}
	this.bodyRead = true
	decoder := json.NewDecoder(this.body)
	decodeErr := decoder.Decode(target)

	if decodeErr == nil {
		_, decodeErr = decoder.Token()

		if decodeErr == io.EOF {
			decodeErr = nil
		} else if decodeErr == nil {
			decodeErr = errors.New("riqu: response body contains additional JSON data")
		}
	}
	closeErr := this.body.Close()
	return errors.Join(decodeErr, closeErr)
}
