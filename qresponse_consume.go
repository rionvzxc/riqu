package riqu

import (
	"errors"
	"fmt"
	"io"
)

// consumeBody centralizes one-shot ownership and transfer completion for every consumer.
func (this *QResponse) consumeBody(consume func(io.Reader) error) (err error) {
	if this == nil {
		return fmt.Errorf("riqu: response is nil")
	}
	if this.isBodyRead {
		return fmt.Errorf("riqu: response body already read")
	}
	if this.body == nil {
		return fmt.Errorf("riqu: response body unavailable")
	}
	this.isBodyRead = true
	isCompleted := false
	defer func() {
		closeErr := this.body.Close()
		if !isCompleted {
			if this.transferBody != nil {
				this.transferBody.finish(errors.New("riqu: response body processing panicked"), false)
			}
			return
		}
		err = errors.Join(err, closeErr)
		if this.transferBody != nil {
			this.transferBody.finish(err, false)
		}
	}()
	err = consume(this.body)
	isCompleted = true
	return err
}
