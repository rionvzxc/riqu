package riqu

import (
 "errors"
 "fmt"
 "io"
)

// consumeBody centralizes one-shot body ownership and cleanup.
func (this *QResponse) consumeBody(consume func(io.Reader) error) (err error) {
 if this == nil { return fmt.Errorf("riqu: response is nil") }
 if this.bodyRead { return fmt.Errorf("riqu: response body already read") }
 if this.body == nil { return fmt.Errorf("riqu: response body unavailable") }
 this.bodyRead = true
 defer func() { err = errors.Join(err, this.body.Close()) }()
 return consume(this.body)
}
