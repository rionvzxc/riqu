package riqu

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type QRequest struct {
	url      string
	method   string
	header   http.Header
	cookies  []*http.Cookie
    cookieJar *QCookieJar
	body     TBody
	timeout  time.Duration
	redirect TRedirect
	configErrors   []error

	onSuccessFunc func(*QResponse)
	onErrorFunc func(error)
}
const (
	VRedirectResponseAll TRedirectResponseMode = iota
	VRedirectResponseLast
)
// Constructor
func New() *QRequest {
	return &QRequest{
		method:  "GET",
		header:  http.Header{},
		cookies:  []*http.Cookie{},
		timeout: 10 * time.Second,
		redirect: TRedirect{
			Max:  10,
			ResponseMode: VRedirectResponseLast,
		},
	}
}
func (this *QRequest) addConfigError(err error) {
	if err != nil {
		this.configErrors = append(this.configErrors, err)
	}
}
func (this *QRequest) Url(url string) *QRequest {
	url = strings.TrimSpace(url)
	this.url = url
	return this
}
func (this *QRequest) Method(method string) *QRequest {
	method = strings.TrimSpace(method)
	this.method = strings.ToUpper(method)
	return this
}
func (this *QRequest) GET(url string) *QRequest {
	return this.Url(url).Method("GET")
}
func (this *QRequest) POST(url string) *QRequest {
	return this.Url(url).Method("POST")
}
func (this *QRequest) PUT(url string) *QRequest {
	return this.Url(url).Method("PUT")
}
func (this *QRequest) DELETE(url string) *QRequest {
	return this.Url(url).Method("DELETE")
}
func (this *QRequest) PATCH(url string) *QRequest {
	return this.Url(url).Method("PATCH")
}
func (this *QRequest) Header(header THeader) *QRequest {
	this.header = header.Clone()
	return this
}
func (this *QRequest) Cookie(cookieMap TValue) *QRequest {
	this.cookies = nil

	for key, val := range cookieMap {
		this.cookies = append(this.cookies, &http.Cookie{
			Name:  key,
			Value: val,
		})
	}
	return this
}
func (this *QRequest) CookieJar(cookieJar *QCookieJar) *QRequest {
	this.cookieJar = cookieJar
    return this
}
func (this *QRequest) Data(data []byte) *QRequest {
	this.body = TBody{
		Data: data,
	}
	return this
}
func (this *QRequest) Text(text string) *QRequest {
	return this.Data([]byte(text))
}
func (this *QRequest) JSON(jsonData any) *QRequest {
	data, err := json.Marshal(jsonData)

	if err != nil {
		this.addConfigError(err)
		return this
	}
	this.body = TBody{
		Data:        data,
		ContentType: "application/json",
	}
	return this
}
func (this *QRequest) Form(formMap TValue) *QRequest {
	form := url.Values{}

	for key, val := range formMap {
		form.Set(key, val)
	}
	this.body = TBody{
		Data:        []byte(form.Encode()),
		ContentType: "application/x-www-form-urlencoded",
	}
	return this
}
func (this *QRequest) FormMultipart(formMap TValue, fileMap TValue) *QRequest {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	// TEXT FIELDS
	for key, val := range formMap {
		if err := writer.WriteField(key, val); err != nil {
			this.addConfigError(err)
			return this
		}
	}
	// FILE FIELDS
	for key, path := range fileMap {
		file, err := os.Open(path)

		if err != nil {
			this.addConfigError(err)
			return this
		}
		part, err := writer.CreateFormFile(key, filepath.Base(path))

		if err != nil {
			file.Close()
			this.addConfigError(err)
			return this
		}
		_, err = io.Copy(part, file)
		file.Close()

		if err != nil {
			this.addConfigError(err)
			return this
		}
	}
	if err := writer.Close(); err != nil {
		this.addConfigError(err)
		return this
	}
	this.body = TBody{
		Data:        buffer.Bytes(),
		ContentType: writer.FormDataContentType(),
	}
	return this
}
func (this *QRequest) Timeout(timeout time.Duration) *QRequest {
	this.timeout = timeout
	return this
}
func (this *QRequest) Redirect(options TRedirect) *QRequest {
	if options.Max < 0 {
		this.addConfigError(fmt.Errorf("Redirect Max cannot be negative"))
		return this
	}
	if options.ResponseMode > VRedirectResponseLast {
		this.addConfigError(fmt.Errorf("Invalid redirect response mode"))
		return this
	}
	this.redirect = options
	return this
}
func (this *QRequest) OnSuccess(callbackFunc func(*QResponse)) *QRequest {
	this.onSuccessFunc = callbackFunc
	return this
}
func (this *QRequest) OnError(callbackFunc func(error)) *QRequest {
	this.onErrorFunc = callbackFunc
	return this
}
func (this *QRequest) handleResponse(response *QResponse) {
	if this.onSuccessFunc != nil {
		this.onSuccessFunc(response)
	}
	response.close()
}
func (this *QRequest) handleResponseError(err error) {
	if this.onErrorFunc != nil {
		this.onErrorFunc(err)
	}
}
func (this *QRequest) Send() {
	if this.url == "" {
		this.addConfigError(fmt.Errorf("Request Url is empty"))
	}
	if this.method == "" {
		this.addConfigError(fmt.Errorf("Request Method is empty"))
	}
	if len(this.configErrors) > 0 {
		fmt.Println("Request Configuration Error:")
        fmt.Println(errors.Join(this.configErrors...))
		return
    }
	httpClient := &http.Client{
		Timeout: this.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	if this.cookieJar != nil {
		httpClient.Jar = this.cookieJar.jar
	}
	currentUrl := this.url
	previousUrl := ""
	currentMethod := this.method
	redirectCount := 0

	for {
		// REQUEST
		var bodyReader io.Reader

		if currentMethod != "GET" && currentMethod != "HEAD" {
			bodyReader = bytes.NewReader(this.body.Data)
		}
		httpReq, err := http.NewRequest(currentMethod, currentUrl, bodyReader)

		if err != nil {
			this.handleResponseError(err)
			return
		}
		// HEADER - First Request
		if redirectCount == 0 {
			httpReq.Header = http.Header(this.header.Clone())

			for _, cookie := range this.cookies {
				httpReq.AddCookie(cookie)
			}
		// HEADER - Redirect
		} else {
			previousUrl, err := url.Parse(previousUrl)

			if err != nil {
				this.handleResponseError(err)
				return
			}
			currentParsed, err := url.Parse(currentUrl)

			if err != nil {
				this.handleResponseError(err)
				return
			}
			sameOrigin := strings.EqualFold(previousUrl.Scheme, currentParsed.Scheme) &&
				strings.EqualFold(previousUrl.Host, currentParsed.Host)
		
			if sameOrigin {
				httpReq.Header = http.Header(this.header.Clone())
			} else {
				allowedHeaders := map[string]bool{
					"Accept":          true,
					"Accept-Language": true,
					"User-Agent":      true,
					"Cache-Control":   true,
					"Pragma":          true,
					"Content-Type":    true,
				}
				for key, values := range this.header {
					canonicalKey := http.CanonicalHeaderKey(key)
		
					if !allowedHeaders[canonicalKey] {
						continue
					}
					for _, value := range values {
						httpReq.Header.Add(canonicalKey, value)
					}
				}
			}
		}
		// HEADER - Content-Type
		if bodyReader != nil && this.body.ContentType != "" {
			httpReq.Header.Set("Content-Type", this.body.ContentType)
		} else if bodyReader == nil {
			httpReq.Header.Del("Content-Type")
		}
		// RESPONSE
		httpRes, err := httpClient.Do(httpReq)

		if err != nil {
			this.handleResponseError(err)
			return
		}
		qRes := &QResponse{
			StatusCode:    httpRes.StatusCode,
			Status:        httpRes.Status,
			Header:        httpRes.Header.Clone(),
			Cookies:       httpRes.Cookies(),
			RequestUrl:    currentUrl,
			RequestMethod: currentMethod,
			body:          httpRes.Body,
		}
		if this.redirect.ResponseMode == VRedirectResponseAll {
			this.handleResponse(qRes)
		}
		follow := redirectCount < this.redirect.Max &&
			IsRedirectStatus(httpRes.StatusCode) &&
			httpRes.Header.Get("Location") != ""
		
		if !follow {
			if this.redirect.ResponseMode == VRedirectResponseLast {
				this.handleResponse(qRes)
			}
			return
		}
		if this.redirect.ResponseMode == VRedirectResponseLast {
			qRes.close()
		}
		// REDIRECT - Set Url & Method
		redirectUrl, err := url.Parse(httpRes.Header.Get("Location"))

		if err != nil {
			this.handleResponseError(err)
			return
		}
		currentParsed, err := url.Parse(currentUrl)

		if err != nil {
			this.handleResponseError(err)
			return
		}
		previousUrl = currentUrl
		currentUrl = currentParsed.ResolveReference(redirectUrl).String()

		switch httpRes.StatusCode {
		case 303:
			if currentMethod != "HEAD" {
				currentMethod = "GET"
			}
		case 301, 302:
			if currentMethod == "POST" {
				currentMethod = "GET"
			}
		}
		redirectCount++
	}
}