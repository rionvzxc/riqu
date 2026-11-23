package riqu

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// VRedirectResponseAll delivers every redirect response to the response callback.
	VRedirectResponseAll TRedirectResponseMode = iota
	// VRedirectResponseLast delivers only the final response to the response callback.
	VRedirectResponseLast
)

type QRequest struct {
	url       string
	method    string
	header    THeader
	cookies   TCookies
	cookieJar *QCookieJar
	body      tRequestBody
	timeout   time.Duration
	redirect  TRedirect
	query     TValues
	querySet  map[string]bool
	queryAll  bool
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
		cookies: TCookies{},
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
// Transport sets the reusable transport used by the request; nil uses the default.
func (this *QRequest) Transport(transport *QTransport) *QRequest {
	this.transport = transport
	return this
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

	if callbackFunc := this.onStatusFunc[response.StatusCode()]; callbackFunc != nil {
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
		configErrors = append(configErrors, errors.New("riqu: timeout cannot be negative"))
	}
	if this.redirect.Max < 0 {
		configErrors = append(configErrors, errors.New("riqu: redirect max cannot be negative"))
	}
	if this.redirect.ResponseMode > VRedirectResponseLast {
		configErrors = append(configErrors, errors.New("riqu: invalid redirect response mode"))
	}
	if this.url == "" {
		configErrors = append(configErrors, errors.New("riqu: request URL is empty"))
	}
	if this.method == "" {
		configErrors = append(configErrors, errors.New("riqu: request method is empty"))
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

	if this.queryAll || len(this.query) > 0 {
		query := make(url.Values)
		if !this.queryAll {
			query, err = url.ParseQuery(initialUrl.RawQuery)
			if err != nil {
				this.handleResponseError(err)
				return err
			}
		}
		for key, values := range this.query {
			if this.querySet[key] {
				delete(query, key)
			}
			query[key] = append(query[key], values...)
		}
		initialUrl.RawQuery = query.Encode()
		initialUrl.ForceQuery = false
		currentUrl = initialUrl.String()
	}
	currentMethod := this.method
	sendBody := currentMethod != "GET" && currentMethod != "HEAD" ||
		this.body.bodyType != vRequestBodyData || preparedBody.data != nil
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
			statusCode:    httpRes.StatusCode,
			status:        httpRes.Status,
			contentLength: httpRes.ContentLength,
			resHeader:     THeader(httpRes.Header).Clone(),
			resCookie:     httpRes.Cookies(),
			resUrl:        currentUrl,
			resMethod:     currentMethod,
			body:          httpRes.Body,
		}

		if this.redirect.ResponseMode == VRedirectResponseAll {
			if err := this.handleResponse(qRes); err != nil {
				this.handleResponseError(err)
				return err
			}
		}
		follow := redirectCount < this.redirect.Max &&
			isRedirectStatus(httpRes.StatusCode) &&
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
