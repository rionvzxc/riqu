package riqu

// SetQuery replaces values for name, preserving other query keys.
func (this *QRequest) SetQuery(name string, values ...string) *QRequest {
	if this.query == nil {
		this.query = make(TValues)
		this.querySet = make(map[string]bool)
	}
	this.query[name] = append([]string(nil), values...)
	this.querySet[name] = true
	return this
}
// AddQuery appends query values to those already configured or present in the URL.
func (this *QRequest) AddQuery(name string, values ...string) *QRequest {
	if this.query == nil {
		this.query = make(TValues)
		this.querySet = make(map[string]bool)
	}
	this.query[name] = append(this.query[name], values...)
	return this
}
// SetAllQueries replaces all query parameters, including those in the URL.
// Nil or empty values clear the entire query.
func (this *QRequest) SetAllQueries(values TValues) *QRequest {
	this.query = values.Clone()
	this.querySet = make(map[string]bool)
	this.queryAll = true
	return this
}
