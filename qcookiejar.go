package riqu

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type QCookieJar struct {
	jar http.CookieJar
}
func NewCookieJar(options *TJarOptions) (*QCookieJar, error) {
	jar, err := cookiejar.New(options)

	if err != nil {
		return nil, err
	}
	return &QCookieJar{jar: jar}, nil
}
func (this *QCookieJar) Cookies(rawURL string) ([]*TCookie, error) {
	if this == nil || this.jar == nil {
		return nil, nil
	}
	u, err := url.Parse(rawURL)

	if err != nil {
		return nil, err
	}
	return this.jar.Cookies(u), nil
}
func (this *QCookieJar) SetCookies(rawURL string, cookies []*TCookie) error {
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