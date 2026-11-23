package riqu

// UserAgent replaces the User-Agent request header.
func (this *QRequest) UserAgent(value string) *QRequest {
	return this.SetHeader("User-Agent", value)
}

// ContentType replaces the Content-Type request header.
func (this *QRequest) ContentType(value string) *QRequest {
	return this.SetHeader("Content-Type", value)
}
