package riqu

// SetCookie replaces the request cookie values for name.
func (this *QRequest) SetCookie(name string, values ...string) *QRequest {
	this.cookies.Del(name)
	return this.AddCookie(name, values...)
}
// AddCookie appends request cookie values for name.
func (this *QRequest) AddCookie(name string, values ...string) *QRequest {
	for _, value := range values {
		this.cookies.Add(TCookie{Name: name, Value: value})
	}
	return this
}
// SetAllCookies replaces all configured request cookie values.
func (this *QRequest) SetAllCookies(values TValues) *QRequest {
	this.cookies = nil

	for name, items := range values {
		this.AddCookie(name, items...)
	}
	return this
}
// CookieJar sets the cookie jar used by the request.
func (this *QRequest) CookieJar(cookieJar *QCookieJar) *QRequest {
	this.cookieJar = cookieJar
	return this
}
