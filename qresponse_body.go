package riqu

import (
	"encoding/json"
	"errors"
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
