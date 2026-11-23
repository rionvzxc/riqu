package riqu

import "fmt"

// Cookie returns a copy of the first response cookie for name, or nil when absent.
func (this *QResponse) Cookie(name string) *TCookie {
	return this.AllCookies().Get(name)
}
// Cookies returns copies of all response cookies matching name.
func (this *QResponse) Cookies(name string) TCookies {
	return this.AllCookies().GetAll(name)
}
// AllCookies returns copies of all cookies set by the response.
func (this *QResponse) AllCookies() TCookies {
	if this == nil {
		return nil
	}
	return this.resCookie.Clone()
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
