package riqu

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

// QCookieJar wraps a standard library HTTP cookie jar.
type QCookieJar struct {
	jar http.CookieJar
}
// NewCookieJar creates a cookie jar with the supplied options.
func NewCookieJar(options *TJarOptions) (*QCookieJar, error) {
	jar, err := cookiejar.New(options)

	if err != nil {
		return nil, err
	}
	return &QCookieJar{jar: jar}, nil
}
// Cookies returns cookies applicable to rawURL.
func (this *QCookieJar) Cookies(rawURL string) (TCookies, error) {
	if this == nil || this.jar == nil {
		return nil, nil
	}
	u, err := url.Parse(rawURL)

	if err != nil {
		return nil, err
	}
	return this.jar.Cookies(u), nil
}
// SetCookies stores cookies for rawURL.
func (this *QCookieJar) SetCookies(rawURL string, cookies TCookies) error {
	if this == nil || this.jar == nil {
		return nil
	}
	u, err := url.Parse(rawURL)

	if err != nil {
		return err
	}
	this.jar.SetCookies(u, cookies)
	return nil
}
