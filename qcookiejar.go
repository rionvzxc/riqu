package riqu

import (
	"net/http"
	"net/url"
)

type QCookieJar struct {
	jar http.CookieJar
}
func (this *QCookieJar) GetCookies(rawURL string) []*TCookie {
	if this == nil || this.jar == nil {
		return nil
	}
	u, err := url.Parse(rawURL)

	if err != nil {
		return nil
	}
	return this.jar.Cookies(u)
}
func (this *QCookieJar) SetCookies(rawURL string, cookies []*TCookie) {
	if this == nil || this.jar == nil {
		return
	}
	u, err := url.Parse(rawURL)
	
	if err != nil {
		return
	}
	this.jar.SetCookies(u, cookies)
}