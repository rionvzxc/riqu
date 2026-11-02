package rxquest

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type QCookieJar struct {
	jar http.CookieJar
}
func NewCookieJar(options *TCookieJarOptions) *QCookieJar {
	var jarOptions *cookiejar.Options

	if options != nil {
		jarOptions = (*cookiejar.Options)(options)
	}
	jar, err := cookiejar.New(jarOptions)

	if err != nil {
		panic(err)
	}
	return &QCookieJar{
		jar: jar,
	}
}
func (this *QCookieJar) Jar() http.CookieJar {
	return this.jar
}
func (this *QCookieJar) Cookies(rawURL string) []*TCookie {
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