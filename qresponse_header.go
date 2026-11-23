package riqu

import "fmt"

// Header returns the first response header value or defaultValue when absent.
func (this *QResponse) Header(name string, defaultValue string) string {
	if this == nil {
		return defaultValue
	}
	return this.resHeader.GetDefault(name, defaultValue)
}
// HeaderValues returns a copy of all response header values for name.
func (this *QResponse) HeaderValues(name string) []string {
	if this == nil {
		return nil
	}
	return this.resHeader.GetAll(name)
}
// AllHeaders returns a clone of all response headers.
func (this *QResponse) AllHeaders() THeader {
	if this == nil {
		return nil
	}
	return this.resHeader.Clone()
}
// BindHeader binds response headers to struct fields using header tags.
func (this *QResponse) BindHeader(target any) error {
	if this == nil {
		return fmt.Errorf("riqu: response is nil")
	}
	return bindValues(target, "header", TValues(this.resHeader))
}
