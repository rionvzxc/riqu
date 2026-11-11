package riqu

import (
	"net/http"
	"net/http/cookiejar"
)

type TMap = map[string]any
type TValue map[string]string
type TJarOptions = cookiejar.Options
type THeader = http.Header
type TCookie = http.Cookie
type TRedirectResponseMode uint
type TRedirect struct {
	Max  int
	ResponseMode TRedirectResponseMode
}
type TBody struct {
	Data        []byte
	ContentType string
}
type TFileOptions struct {
	Dir             string
	Filename        string
	DefaultFilename string
}