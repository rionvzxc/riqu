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

type requestBodyKind uint

const (
	requestBodyData requestBodyKind = iota
	requestBodyText
	requestBodyHTML
	requestBodyJSON
	requestBodyForm
	requestBodyMultipart
)

const (
	// VRedirectResponseAll delivers every redirect response to the success callback.
	VRedirectResponseAll TRedirectResponseMode = iota
	// VRedirectResponseLast delivers only the final response to the success callback.
	VRedirectResponseLast
)

type preparedBody struct {
	data        []byte
	contentType string
	multipart   *TMultipart
}
type QRequest struct {
	url       string
	method    string
	header    THeader
	cookies   []*http.Cookie
	cookieJar *QCookieJar
	body      TBody
	bodyKind  requestBodyKind
	bodyValue any
	timeout   time.Duration
	redirect  TRedirect
	query     TValues
	querySet  map[string]bool
	transport *QTransport

	onStatusFunc  map[int]func(*QResponse)
	onSuccessFunc func(*QResponse)
	onErrorFunc   func(error)
}
// New creates a request with default method, timeout, and redirect settings.
func New() *QRequest {
	return &QRequest{
		method:  "GET",
		header:  THeader{},
		cookies: []*http.Cookie{},
		timeout: 10 * time.Second,
		redirect: TRedirect{
			Max:          10,
			ResponseMode: VRedirectResponseLast,
		},
	}
}
// Url sets the request URL.
func (this *QRequest) Url(url string) *QRequest {
	url = strings.TrimSpace(url)
	this.url = url
	return this
}
// Query replaces values for the supplied query keys, preserving other URL keys.
func (this *QRequest) Query(values TValues) *QRequest {
	if this.query == nil {
		this.query = make(TValues)
		this.querySet = make(map[string]bool)
	}
	for key, items := range values {
		this.query[key] = append([]string(nil), items...)
		this.querySet[key] = true
	}
	return this
}
// AddQuery appends query values to those already configured or present in the URL.
func (this *QRequest) AddQuery(values TValues) *QRequest {
	if this.query == nil {
		this.query = make(TValues)
		this.querySet = make(map[string]bool)
	}
	for key, items := range values {
		this.query[key] = append(this.query[key], items...)
	}
	return this
}
// Method sets the HTTP method.
func (this *QRequest) Method(method string) *QRequest {
	method = strings.TrimSpace(method)
	this.method = strings.ToUpper(method)
	return this
}
// GET configures a GET request for url.
func (this *QRequest) GET(url string) *QRequest {
	return this.Url(url).Method("GET")
}
// POST configures a POST request for url.
func (this *QRequest) POST(url string) *QRequest {
	return this.Url(url).Method("POST")
}
// PUT configures a PUT request for url.
func (this *QRequest) PUT(url string) *QRequest {
	return this.Url(url).Method("PUT")
}
// DELETE configures a DELETE request for url.
func (this *QRequest) DELETE(url string) *QRequest {
	return this.Url(url).Method("DELETE")
}
// PATCH configures a PATCH request for url.
func (this *QRequest) PATCH(url string) *QRequest {
	return this.Url(url).Method("PATCH")
}
// Header sets the request headers.
func (this *QRequest) Header(header THeader) *QRequest {
	if header == nil {
		this.header = make(THeader)
	} else {
		this.header = header.Clone()
	}
	return this
}
// Cookie sets request cookies from cookieMap.
func (this *QRequest) Cookie(cookieMap TValues) *QRequest {
	this.cookies = nil

	for key := range cookieMap {
		this.cookies = append(this.cookies, &http.Cookie{
			Name:  key,
			Value: cookieMap.Get(key),
		})
	}
	return this
}
// CookieJar sets the cookie jar used by the request.
func (this *QRequest) CookieJar(cookieJar *QCookieJar) *QRequest {
	this.cookieJar = cookieJar
	return this
}
// Transport sets the reusable transport used by the request; nil uses the default.
func (this *QRequest) Transport(transport *QTransport) *QRequest {
	this.transport = transport
	return this
}
// Data sets a raw request body.
func (this *QRequest) Data(data []byte) *QRequest {
	this.body = TBody{
		Data: data,
	}
	this.bodyKind = requestBodyData
	this.bodyValue = nil
	return this
}
// Text sets a plain text request body.
func (this *QRequest) Text(text string) *QRequest {
	this.body = TBody{
		ContentType: "text/plain",
	}
	this.bodyKind = requestBodyText
	this.bodyValue = text
	return this
}
// HTML sets an HTML request body.
func (this *QRequest) HTML(html string) *QRequest {
	this.body = TBody{
		ContentType: "text/html",
	}
	this.bodyKind = requestBodyHTML
	this.bodyValue = html
	return this
}
// JSON sets a request body that is JSON-encoded when sent.
func (this *QRequest) JSON(jsonData any) *QRequest {
	this.body = TBody{
		ContentType: "application/json",
	}
	this.bodyKind = requestBodyJSON
	this.bodyValue = jsonData
	return this
}
// Form sets a URL-encoded form request body.
func (this *QRequest) Form(formMap TValues) *QRequest {
	this.body = TBody{
		ContentType: "application/x-www-form-urlencoded",
	}
	this.bodyKind = requestBodyForm
	this.bodyValue = formMap.Clone()
	return this
}
// FormMultipart sets a multipart form body with fields and file paths.
func (this *QRequest) FormMultipart(multipart TMultipart) *QRequest {
	this.body = TBody{}
	this.bodyKind = requestBodyMultipart
	this.bodyValue = TMultipart{
		Form: multipart.Form.Clone(),
		File: multipart.File.Clone(),
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
		formMap, ok := this.bodyValue.(TValues)

		if !ok {
			return preparedBody{}, errors.New("Invalid form body")
		}
		form := url.Values{}

		for key, value := range formMap {
			form[key] = append([]string(nil), value...)
		}
		prepared.data = []byte(form.Encode())
	case requestBodyMultipart:
		formData, ok := this.bodyValue.(TMultipart)

		if !ok {
			return preparedBody{}, errors.New("Invalid multipart body")
		}
		prepared.multipart = &formData
	default:
		return preparedBody{}, errors.New("Invalid request body type")
	}
	return prepared, nil
}
func (this *QRequest) multipartBodyReader(formData TMultipart) (io.ReadCloser, string, error) {
	contentType := this.header.Get("Content-Type")
	var params map[string]string

	if contentType != "" {
		mediaType, parsedParams, err := mime.ParseMediaType(contentType)

		if err != nil {
			return nil, "", fmt.Errorf("invalid multipart Content-Type: %w", err)
		}
		if mediaType != "multipart/form-data" {
			return nil, "", fmt.Errorf("invalid multipart Content-Type %q: expected multipart/form-data", mediaType)
		}
		params = parsedParams
	}
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)

	if contentType == "" {
		contentType = writer.FormDataContentType()
	} else if params["boundary"] == "" {
		params["boundary"] = writer.Boundary()
		contentType = mime.FormatMediaType("multipart/form-data", params)
	} else if err := writer.SetBoundary(params["boundary"]); err != nil {
		pipeReader.Close()
		pipeWriter.CloseWithError(err)
		return nil, "", err
	}

	go func() {
		var writeErr error

		for key, values := range formData.Form {
			for _, value := range values {
				if err := writer.WriteField(key, value); err != nil {
					writeErr = err
					break
				}
			}
			if writeErr != nil {
				break
			}
		}
		if writeErr == nil {
			for key, paths := range formData.File {
				for _, path := range paths {
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
				if writeErr != nil {
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
// Timeout sets the request timeout; zero disables the timeout.
func (this *QRequest) Timeout(timeout time.Duration) *QRequest {
	this.timeout = timeout
	return this
}
// Redirect sets the redirect policy for the request.
func (this *QRequest) Redirect(options TRedirect) *QRequest {
	this.redirect = options
	return this
}
// OnSuccess sets the fallback callback for responses without a status handler.
func (this *QRequest) OnSuccess(callbackFunc func(*QResponse)) *QRequest {
	this.onSuccessFunc = callbackFunc
	return this
}
// OnStatus sets a status handler, replacing the previous handler for that status.
// A nil handler restores the OnSuccess fallback.
func (this *QRequest) OnStatus(status int, callbackFunc func(*QResponse)) *QRequest {
	if this.onStatusFunc == nil {
		this.onStatusFunc = make(map[int]func(*QResponse))
	}
	this.onStatusFunc[status] = callbackFunc
	return this
}
// OnStatuses sets the same handler for each supplied status.
func (this *QRequest) OnStatuses(statuses []int, callbackFunc func(*QResponse)) *QRequest {
	for _, status := range statuses {
		this.OnStatus(status, callbackFunc)
	}
	return this
}
// OnError sets the callback invoked when request processing fails.
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

	if callbackFunc := this.onStatusFunc[response.StatusCode]; callbackFunc != nil {
		callbackFunc(response)
	} else if this.onSuccessFunc != nil {
		this.onSuccessFunc(response)
	}
	return nil
}
func (this *QRequest) handleResponseError(err error) {
	if this.onErrorFunc != nil {
		this.onErrorFunc(err)
	}
}
// Send executes the configured HTTP request.
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
	if this.transport != nil && this.transport.transport != nil {
		httpClient.Transport = this.transport.transport
	}
	initialUrl, err := url.Parse(this.url)

	if err != nil {
		this.handleResponseError(err)
		return err
	}
	currentUrl := this.url

	if len(this.query) > 0 {
		query, err := url.ParseQuery(initialUrl.RawQuery)

		if err != nil {
			this.handleResponseError(err)
			return err
		}
		for key, values := range this.query {
			if this.querySet[key] {
				delete(query, key)
			}
			query[key] = append(query[key], values...)
		}
		initialUrl.RawQuery = query.Encode()
		currentUrl = initialUrl.String()
	}
	currentMethod := this.method
	sendBody := currentMethod != "GET" && currentMethod != "HEAD" ||
		this.bodyKind != requestBodyData || this.body.Data != nil
	redirectCount := 0
	initialRequest := &http.Request{Header: http.Header(this.header.Clone())}

	for _, cookie := range this.cookies {
		initialRequest.AddCookie(cookie)
	}

	for {
		// REQUEST
		var bodyReader io.Reader
		var bodyCloser io.Closer
		bodyContentType := preparedBody.contentType

		if sendBody {
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
			httpReq.Header = initialRequest.Header.Clone()
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
				httpReq.Header = initialRequest.Header.Clone()
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
		if host := httpReq.Header.Get("Host"); host != "" {
			httpReq.Host = host
		}
		httpReq.Header.Del("Host")
		// HEADER - Content-Type
		if bodyReader == nil {
			httpReq.Header.Del("Content-Type")
		} else if preparedBody.multipart != nil || (!this.header.Exist("Content-Type") && bodyContentType != "") {
			httpReq.Header.Set("Content-Type", bodyContentType)
		}
		// RESPONSE
		httpRes, err := httpClient.Do(httpReq)

		if err != nil {
			this.handleResponseError(err)
			return err
		}
		if httpClient.Jar != nil && len(httpRes.Cookies()) > 0 {
			replacedCookies := make(map[string]bool)

			for _, cookie := range httpRes.Cookies() {
				replacedCookies[cookie.Name] = true
			}
			cookies := initialRequest.Cookies()
			initialRequest.Header.Del("Cookie")

			for _, cookie := range cookies {
				if !replacedCookies[cookie.Name] {
					initialRequest.AddCookie(cookie)
				}
			}
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
			sendBody = false

			if currentMethod != "HEAD" {
				currentMethod = "GET"
			}
		case 301, 302:
			if currentMethod == "POST" {
				currentMethod = "GET"
				sendBody = false
			}
		}
		redirectCount++
	}
}
