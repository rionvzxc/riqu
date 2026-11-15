package riqu

import "net/http"

// THeader stores HTTP header values using canonical header names.
type THeader map[string][]string

// Set replaces the values for an HTTP header.
func (this THeader) Set(name string, values ...string) {
	this[http.CanonicalHeaderKey(name)] = append([]string(nil), values...)
}
// Add appends values to an HTTP header.
func (this THeader) Add(name string, values ...string) {
	canonicalName := http.CanonicalHeaderKey(name)
	this[canonicalName] = append(this[canonicalName], values...)
}
// Get returns the first value for an HTTP header.
func (this THeader) Get(name string) string {
	values := this[http.CanonicalHeaderKey(name)]

	if len(values) == 0 {
		return ""
	}
	return values[0]
}
// GetAll returns all values for an HTTP header.
func (this THeader) GetAll(name string) []string {
	return append([]string(nil), this[http.CanonicalHeaderKey(name)]...)
}
// Exist reports whether an HTTP header is present.
func (this THeader) Exist(name string) bool {
	_, exists := this[http.CanonicalHeaderKey(name)]
	return exists
}
// Del removes an HTTP header.
func (this THeader) Del(name string) {
	delete(this, http.CanonicalHeaderKey(name))
}
// Clone returns an independent copy of the header map.
func (this THeader) Clone() THeader {
	if this == nil {
		return nil
	}
	header := make(THeader, len(this))

	for name, values := range this {
		canonicalName := http.CanonicalHeaderKey(name)

		if values == nil {
			if _, exists := header[canonicalName]; !exists {
				header[canonicalName] = nil
			}
		} else {
			if _, exists := header[canonicalName]; !exists {
				header[canonicalName] = make([]string, 0, len(values))
			}
			header[canonicalName] = append(header[canonicalName], values...)
		}
	}
	return header
}
