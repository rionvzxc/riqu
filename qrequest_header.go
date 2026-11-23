package riqu

// UserAgent replaces the User-Agent request header.
func (this *QRequest) UserAgent(value string) *QRequest {
	return this.SetHeader("User-Agent", value)
}
// SetHeader replaces the request values for name.
func (this *QRequest) SetHeader(name string, values ...string) *QRequest {
	if this.header == nil {
		this.header = make(THeader)
	}
	this.header.Set(name, values...)
	return this
}
// AddHeader appends request values for name.
func (this *QRequest) AddHeader(name string, values ...string) *QRequest {
	if this.header == nil {
		this.header = make(THeader)
	}
	this.header.Add(name, values...)
	return this
}
// SetAllHeaders replaces all request headers with a clone of header.
func (this *QRequest) SetAllHeaders(header THeader) *QRequest {
	this.header = header.Clone()

	if this.header == nil {
		this.header = make(THeader)
	}
	return this
}
