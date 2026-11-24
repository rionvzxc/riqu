package riqu

import (
	"errors"
	"io"
)

type tResponseLimitBody struct {
	io.ReadCloser
	remaining int64
	isExceeded  bool
}

func (this *tResponseLimitBody) Read(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if this.isExceeded {
		return 0, errors.New("riqu: response body exceeds maximum size")
	}
	if this.remaining == 0 {
		var probe [1]byte
		bytesRead, err := this.ReadCloser.Read(probe[:])

		if bytesRead > 0 {
			this.isExceeded = true
			return 0, errors.New("riqu: response body exceeds maximum size")
		}
		return 0, err
	}
	if int64(len(data)) > this.remaining {
		data = data[:this.remaining]
	}
	bytesRead, err := this.ReadCloser.Read(data)
	this.remaining -= int64(bytesRead)
	return bytesRead, err
}
