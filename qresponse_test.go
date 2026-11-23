package riqu

import "testing"

func TestNilResponseMetadata(t *testing.T) {
	var response *QResponse
	if response.StatusCode() != 0 || response.Status() != "" || response.Url() != "" || response.Method() != "" {
		t.Fatal("unexpected nil response metadata")
	}
	if response.Header("missing", "default") != "default" || response.AllHeaders() != nil || response.HeaderValues("x") != nil {
		t.Fatal("unexpected nil response headers")
	}
	if response.Cookie("x") != nil || response.Cookies("x") != nil || response.AllCookies() != nil {
		t.Fatal("unexpected nil response cookies")
	}
	if response.BindHeader(&struct{}{}) == nil || response.BindCookie(&struct{}{}) == nil {
		t.Fatal("binding a nil response must fail")
	}
}
