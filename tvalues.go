package riqu

// TValues stores multiple string values for each key.
type TValues map[string][]string

// Set replaces the values associated with name.
func (this TValues) Set(name string, values ...string) {
	this[name] = append([]string(nil), values...)
}
// Add appends values to those associated with name.
func (this TValues) Add(name string, values ...string) {
	this[name] = append(this[name], values...)
}
// Get returns the first value associated with name.
func (this TValues) Get(name string) string {
	values := this[name]

	if len(values) == 0 {
		return ""
	}
	return values[0]
}
// GetAll returns all values associated with name.
func (this TValues) GetAll(name string) []string {
	return append([]string(nil), this[name]...)
}
// Exist reports whether name is present.
func (this TValues) Exist(name string) bool {
	_, exists := this[name]
	return exists
}
// Del removes name and its values.
func (this TValues) Del(name string) {
	delete(this, name)
}
// Clone returns an independent copy of the values map.
func (this TValues) Clone() TValues {
	if this == nil {
		return nil
	}
	cloned := make(TValues, len(this))

	for name, values := range this {
		if values == nil {
			cloned[name] = nil
		} else {
			cloned[name] = make([]string, len(values))
			copy(cloned[name], values)
		}
	}
	return cloned
}
