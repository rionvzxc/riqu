package riqu

type TValues map[string][]string

func (this TValues) Set(name string, values ...string) {
	this[name] = append([]string(nil), values...)
}
func (this TValues) Add(name string, values ...string) {
	this[name] = append(this[name], values...)
}
func (this TValues) Get(name string) string {
	values := this[name]

	if len(values) == 0 {
		return ""
	}
	return values[0]
}
func (this TValues) GetAll(name string) []string {
	return append([]string(nil), this[name]...)
}
func (this TValues) Exist(name string) bool {
	_, exists := this[name]
	return exists
}
func (this TValues) Del(name string) {
	delete(this, name)
}
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
