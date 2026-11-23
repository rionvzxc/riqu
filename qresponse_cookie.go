package riqu

import "fmt"

// Cookie returns a copy of the first response cookie for name, or nil when absent.
func (this *QResponse) Cookie(name string) *TCookie {
	for _, cookie := range this.AllCookies() {
		if cookie != nil && cookie.Name == name {
			return cookie
		}
	}
	return nil
}
// Cookies returns copies of all response cookies matching name.
func (this *QResponse) Cookies(name string) []*TCookie {
	var cookies []*TCookie

	for _, cookie := range this.AllCookies() {
		if cookie != nil && cookie.Name == name {
			cookies = append(cookies, cookie)
		}
	}
	return cookies
}
// AllCookies returns copies of all cookies set by the response.
func (this *QResponse) AllCookies() []*TCookie {
	if this == nil {
		return nil
	}
	if this.resCookie == nil {
		return nil
	}
	cookies := make([]*TCookie, len(this.resCookie))

	for i, cookie := range this.resCookie {
		if cookie == nil {
			continue
		}
		clone := *cookie
		clone.Unparsed = append([]string(nil), cookie.Unparsed...)
		cookies[i] = &clone
	}
	return cookies
}
// BindCookie binds response cookie values to struct fields using cookie tags.
func (this *QResponse) BindCookie(target any) error {
	if this == nil {
		return fmt.Errorf("riqu: response is nil")
	}
	values := make(TValues)

	for _, cookie := range this.resCookie {
		if cookie != nil {
			values.Add(cookie.Name, cookie.Value)
		}
	}
	return bindValues(target, "cookie", values)
}
