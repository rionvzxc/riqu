package riqu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
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
	bodyKind  requestBodyKind
	bodyValue any
	timeout  time.Duration
	redirect TRedirect

	onSuccessFunc func(*QResponse)
	onErrorFunc func(error)
}
type requestBodyKind uint

const (
	requestBodyData requestBodyKind = iota
	requestBodyText
	requestBodyHTML
	requestBodyJSON
	requestBodyForm
	requestBodyMultipart
)

type multipartBody struct {
	fields TValue
	files  TValue
}

type preparedBody struct {
	data        []byte
	contentType string
	multipart   *multipartBody
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
	if header == nil {
		this.header = make(THeader)
	} else {
		this.header = header.Clone()
	}
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
func cloneValues(values TValue) TValue {
	if values == nil {
		return nil
	}
	cloned := make(TValue, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
func hasHeader(header http.Header, name string) bool {
	for key := range header {
		if strings.EqualFold(key, name) {
			return true
		}
	}
	return false
}
func getHeaderValue(header http.Header, name string) (string, bool) {
	for key, values := range header {
		if !strings.EqualFold(key, name) {
			continue
		}
		if len(values) == 0 {
			return "", true
		}
		return values[0], true
	}
	return "", false
}
func (this *QRequest) Data(data []byte) *QRequest {
	this.body = TBody{
		Data: data,
	}
	this.bodyKind = requestBodyData
	this.bodyValue = nil
	return this
}
func (this *QRequest) Text(text string) *QRequest {
	this.body = TBody{
		ContentType: "text/plain",
	}
	this.bodyKind = requestBodyText
	this.bodyValue = text
	return this
}
func (this *QRequest) HTML(html string) *QRequest {
	this.body = TBody{
		ContentType: "text/html",
	}
	this.bodyKind = requestBodyHTML
	this.bodyValue = html
	return this
}
func (this *QRequest) JSON(jsonData any) *QRequest {
	this.body = TBody{
		ContentType: "application/json",
	}
	this.bodyKind = requestBodyJSON
	this.bodyValue = jsonData
	return this
}
func (this *QRequest) Form(formMap TValue) *QRequest {
	this.body = TBody{
		ContentType: "application/x-www-form-urlencoded",
	}
	this.bodyKind = requestBodyForm
	this.bodyValue = cloneValues(formMap)
	return this
}

func (this *QRequest) FormMultipart(formMap TValue, fileMap TValue) *QRequest {
	this.body = TBody{}
	this.bodyKind = requestBodyMultipart
	this.bodyValue = multipartBody{
		fields: cloneValues(formMap),
		files:  cloneValues(fileMap),
	}
	return this
}
func (this *QRequest) prepareBody() (preparedBody, error) {
	prepared := preparedBody{contentType: this.body.ContentType}
	switch this.bodyKind {
	case requestBodyData:
		prepared.data = this.body.Data
	case requestBodyText, requestBodyHTML:
		value, ok := this.bodyValue.(string)
		if !ok {
			return preparedBody{}, errors.New("Invalid request body")
		}
		prepared.data = []byte(value)
	case requestBodyJSON:
		data, err := json.Marshal(this.bodyValue)
		if err != nil {
			return preparedBody{}, err
		}
		prepared.data = data
	case requestBodyForm:
		formMap, ok := this.bodyValue.(TValue)
		if !ok {
			return preparedBody{}, errors.New("Invalid form body")
		}
		form := url.Values{}
		for key, value := range formMap {
			form.Set(key, value)
		}
		prepared.data = []byte(form.Encode())
	case requestBodyMultipart:
		formData, ok := this.bodyValue.(multipartBody)
		if !ok {
			return preparedBody{}, errors.New("Invalid multipart body")
		}
		prepared.multipart = &formData
	default:
		return preparedBody{}, errors.New("Invalid request body type")
	}
	return prepared, nil
}
func (this *QRequest) multipartBodyReader(formData multipartBody) (io.ReadCloser, string, error) {
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	contentType, hasContentType := getHeaderValue(this.header, "Content-Type")
	if hasContentType {
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err == nil && strings.EqualFold(mediaType, "multipart/form-data") && params["boundary"] != "" {
			if err := writer.SetBoundary(params["boundary"]); err != nil {
				pipeReader.Close()
				pipeWriter.CloseWithError(err)
				return nil, "", err
			}
		}
	} else {
		contentType = writer.FormDataContentType()
	}

	go func() {
		var writeErr error
		for key, value := range formData.fields {
			if err := writer.WriteField(key, value); err != nil {
				writeErr = err
				break
			}
		}
		if writeErr == nil {
			for key, path := range formData.files {
				file, err := os.Open(path)
				if err != nil {
					writeErr = fmt.Errorf("open multipart file %q: %w", path, err)
					break
				}
				part, err := writer.CreateFormFile(key, filepath.Base(path))
				if err != nil {
					writeErr = errors.Join(fmt.Errorf("create multipart file %q: %w", path, err), file.Close())
					break
				}
				_, copyErr := io.Copy(part, file)
				closeErr := file.Close()
				if copyErr != nil {
					copyErr = fmt.Errorf("copy multipart file %q: %w", path, copyErr)
				}
				if closeErr != nil {
					closeErr = fmt.Errorf("close multipart file %q: %w", path, closeErr)
				}
				if err := errors.Join(copyErr, closeErr); err != nil {
					writeErr = err
					break
				}
			}
		}
		if writeErr == nil {
			writeErr = writer.Close()
		}
		if writeErr != nil {
			pipeWriter.CloseWithError(writeErr)
			return
		}
		pipeWriter.Close()
	}()
	return pipeReader, contentType, nil
}
func (this *QRequest) Timeout(timeout time.Duration) *QRequest {
	this.timeout = timeout
    return this
}
func (this *QRequest) Redirect(options TRedirect) *QRequest {
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
func (this *QRequest) handleResponse(response *QResponse) (err error) {
	defer func() {
		if !response.bodyRead {
			err = response.close()
		}
	}()
	if this.onSuccessFunc != nil {
		this.onSuccessFunc(response)
	}
	return nil
}
func (this *QRequest) handleResponseError(err error) {
	if this.onErrorFunc != nil {
		this.onErrorFunc(err)
	}
}
func (this *QRequest) Send() error {
	var configErrors []error
	if this.timeout < 0 {
		configErrors = append(configErrors, errors.New("timeout cannot be negative"))
	}
	if this.redirect.Max < 0 {
		configErrors = append(configErrors, errors.New("Redirect Max cannot be negative"))
	}
	if this.redirect.ResponseMode > VRedirectResponseLast {
		configErrors = append(configErrors, errors.New("Invalid redirect response mode"))
	}

	if this.url == "" {
		configErrors = append(configErrors, errors.New("Request Url is empty"))
	}
	if this.method == "" {
		configErrors = append(configErrors, errors.New("Request Method is empty"))
	}
	if err := errors.Join(configErrors...); err != nil {
		this.handleResponseError(err)
		return err
	}
	preparedBody, err := this.prepareBody()
	if err != nil {
		this.handleResponseError(err)
		return err
	}
	requestContext := context.Background()
	if this.timeout > 0 {
		var cancel context.CancelFunc
		requestContext, cancel = context.WithTimeout(requestContext, this.timeout)
		defer cancel()
	}
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	if this.cookieJar != nil {
		httpClient.Jar = this.cookieJar.jar
	}
	initialUrl, err := url.Parse(this.url)
	if err != nil {
		this.handleResponseError(err)
		return err
	}
	currentUrl := this.url
	currentMethod := this.method
	redirectCount := 0

	for {
		// REQUEST
		var bodyReader io.Reader
		var bodyCloser io.Closer
		bodyContentType := preparedBody.contentType

		if currentMethod != "GET" && currentMethod != "HEAD" {
			if preparedBody.multipart != nil {
				multipartReader, contentType, err := this.multipartBodyReader(*preparedBody.multipart)
				if err != nil {
					this.handleResponseError(err)
					return err
				}
				bodyReader = multipartReader
				bodyCloser = multipartReader
				bodyContentType = contentType
			} else {
				bodyReader = bytes.NewReader(preparedBody.data)
			}
		}
		httpReq, err := http.NewRequestWithContext(requestContext, currentMethod, currentUrl, bodyReader)
		if err != nil {
			if bodyCloser != nil {
				bodyCloser.Close()
			}
			this.handleResponseError(err)
			return err
		}
		// HEADER - First Request
		if redirectCount == 0 {
			httpReq.Header = http.Header(this.header.Clone())

			for _, cookie := range this.cookies {
				httpReq.AddCookie(cookie)
			}
		// HEADER - Redirect
		} else {
			currentParsed, err := url.Parse(currentUrl)

			if err != nil {
				this.handleResponseError(err)
				return err
			}
			sameOrigin := strings.EqualFold(initialUrl.Scheme, currentParsed.Scheme) &&
				strings.EqualFold(initialUrl.Host, currentParsed.Host)

			if sameOrigin {
				httpReq.Header = http.Header(this.header.Clone())
				for _, cookie := range this.cookies {
					httpReq.AddCookie(cookie)
				}
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
		if bodyReader == nil {
			httpReq.Header.Del("Content-Type")
		} else if !hasHeader(this.header, "Content-Type") && bodyContentType != "" {
			httpReq.Header.Set("Content-Type", bodyContentType)
		}
		// RESPONSE
		httpRes, err := httpClient.Do(httpReq)

		if err != nil {
			this.handleResponseError(err)
			return err
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
			if err := this.handleResponse(qRes); err != nil {
				this.handleResponseError(err)
				return err
			}
		}
		follow := redirectCount < this.redirect.Max &&
			IsRedirectStatus(httpRes.StatusCode) &&
			httpRes.Header.Get("Location") != ""

		if !follow {
			if this.redirect.ResponseMode == VRedirectResponseLast {
				if err := this.handleResponse(qRes); err != nil {
					this.handleResponseError(err)
					return err
				}
			}
			return nil
		}
		if this.redirect.ResponseMode == VRedirectResponseLast {
			if err := qRes.close(); err != nil {
				this.handleResponseError(err)
				return err
			}
		}
		// REDIRECT - Set Url & Method
		redirectUrl, err := url.Parse(httpRes.Header.Get("Location"))

		if err != nil {
			this.handleResponseError(err)
			return err
		}
		currentParsed, err := url.Parse(currentUrl)

		if err != nil {
			this.handleResponseError(err)
			return err
		}
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
