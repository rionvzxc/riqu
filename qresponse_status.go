package riqu

// IsInformational reports whether the response status is in the range 100–199.
func (this *QResponse) IsInformational() bool {
	return this.StatusCode() >= 100 && this.StatusCode() < 200
}

// IsSuccess reports whether the response status is in the range 200–299.
func (this *QResponse) IsSuccess() bool {
	return this.StatusCode() >= 200 && this.StatusCode() < 300
}

// IsRedirect reports whether the response status is in the range 300–399.
func (this *QResponse) IsRedirect() bool {
	return this.StatusCode() >= 300 && this.StatusCode() < 400
}

// IsClientError reports whether the response status is in the range 400–499.
func (this *QResponse) IsClientError() bool {
	return this.StatusCode() >= 400 && this.StatusCode() < 500
}

// IsServerError reports whether the response status is in the range 500–599.
func (this *QResponse) IsServerError() bool {
	return this.StatusCode() >= 500 && this.StatusCode() < 600
}
