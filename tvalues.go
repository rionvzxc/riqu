package riqu

// TValues stores multiple string values under each name.
type TValues map[string][]string

// Set replaces the values associated with name.
func (this TValues) Set(name string, values ...string) {
	this[name] = append([]string(nil), values...)
}
// Add appends values to those associated with name.
func (this TValues) Add(name string, values ...string) {
	this[name] = append(this[name], values...)
}
// Get returns the first value associated with name, or an empty string.
func (this TValues) Get(name string) string {
	return this.GetDefault(name, "")
}
// GetDefault returns the first value for name, or defaultValue when no values exist.
func (this TValues) GetDefault(name string, defaultValue string) string {
	values := this[name]

	if len(values) == 0 {
		return defaultValue
	}
	return values[0]
}
// GetLast returns the last value associated with name, or an empty string.
func (this TValues) GetLast(name string) string {
	values := this[name]

	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}
// GetAll returns a copy of all values associated with name.
func (this TValues) GetAll(name string) []string {
	return append([]string(nil), this[name]...)
}
// Exist reports whether name is present in the map.
func (this TValues) Exist(name string) bool {
	_, isPresent := this[name]
	return isPresent
}
// Del removes name and its values from the map.
func (this TValues) Del(name string) {
	delete(this, name)
}
// Clone returns a deep copy of the map and its value slices.
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
