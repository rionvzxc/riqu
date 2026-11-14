package riqu

import "testing"

func TestCookieJarHandlesNilReceiver(t *testing.T) {
	var cookieJar *QCookieJar

	cookies, err := cookieJar.Cookies("https://example.test")
	if err != nil {
		t.Fatal(err)
	}
	if cookies != nil {
		t.Fatalf("expected no cookies from nil jar, got %#v", cookies)
	}
	if err := cookieJar.SetCookies("https://example.test", nil); err != nil {
		t.Fatal(err)
	}
}

func TestNewCookieJarAcceptsNilOptions(t *testing.T) {
	cookieJar, err := NewCookieJar(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cookieJar == nil {
		t.Fatal("expected a cookie jar")
	}
}
