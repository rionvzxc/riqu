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
	// VLastResponse delivers only the final response to the response callback.
	VLastResponse TResponseMode = iota
	// VAllResponses delivers every response in the redirect chain to a response callback.
	VAllResponses
)

type QRequest struct {
	maxResponseSize int64
	reqContext context.Context
	url       string
	method    string
	header    THeader
	cookies   TCookies
	cookieJar *QCookieJar
	body      tRequestBody
	timeout   time.Duration
	redirectOptions TRedirectOptions
	query     TValues
	querySet  map[string]bool
	queryAll  bool
	transport *QTransport

	onStatusFunc  map[int]func(*QResponse) error
	onResponseFunc func(*QResponse) error
	onErrorFunc   func(error)
}
// New creates a request with default method, timeout, and redirect settings.
func New() *QRequest {
	return &QRequest{
		method:  "GET",
		header:  THeader{},
		cookies: TCookies{},
		timeout: 10 * time.Second,
		redirectOptions: TRedirectOptions{
			Max:          10,
			Responses: VLastResponse,
		},
	}
}
// URL sets the request URL.
func (this *QRequest) URL(url string) *QRequest {
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
	return this.URL(url).Method("GET")
}
// POST configures a POST request for url.
func (this *QRequest) POST(url string) *QRequest {
	return this.URL(url).Method("POST")
}
// PUT configures a PUT request for url.
func (this *QRequest) PUT(url string) *QRequest {
	return this.URL(url).Method("PUT")
}
// DELETE configures a DELETE request for url.
func (this *QRequest) DELETE(url string) *QRequest {
	return this.URL(url).Method("DELETE")
}
// PATCH configures a PATCH request for url.
func (this *QRequest) PATCH(url string) *QRequest {
	return this.URL(url).Method("PATCH")
}
// Transport sets the reusable transport used by the request; nil uses the default.
func (this *QRequest) Transport(transport *QTransport) *QRequest {
	this.transport = transport
	return this
}
// Context sets the parent context; nil uses context.Background.
// It applies to all redirects and response body reads during Send.
func (this *QRequest) Context(ctx context.Context) *QRequest {
	this.reqContext = ctx
	return this
}
// Timeout sets the request timeout; zero disables the timeout.
func (this *QRequest) Timeout(timeout time.Duration) *QRequest {
	this.timeout = timeout
	return this
}
// MaxResponseSize limits decoded response body bytes; zero disables the limit.
func (this *QRequest) MaxResponseSize(size int64) *QRequest {
	this.maxResponseSize = size
	return this
}
// RedirectOptions sets the redirect policy for the request.
func (this *QRequest) RedirectOptions(options TRedirectOptions) *QRequest {
	this.redirectOptions = options
	return this
}
// OnResponse sets the fallback callback for responses without a status handler.
// Callback errors are passed to OnError and returned by Send.
func (this *QRequest) OnResponse(callbackFunc func(*QResponse) error) *QRequest {
	this.onResponseFunc = callbackFunc
	return this
}
// OnStatus sets a status handler, replacing the previous handler for that status.
// A nil handler restores the OnResponse fallback.
// Callback errors are passed to OnError and returned by Send.
func (this *QRequest) OnStatus(status int, callbackFunc func(*QResponse) error) *QRequest {
	if this.onStatusFunc == nil {
		this.onStatusFunc = make(map[int]func(*QResponse) error)
	}
	this.onStatusFunc[status] = callbackFunc
	return this
}
// OnStatuses sets the same handler for each supplied status.
// Callback errors are passed to OnError and returned by Send.
func (this *QRequest) OnStatuses(statuses []int, callbackFunc func(*QResponse) error) *QRequest {
	for _, status := range statuses {
		this.OnStatus(status, callbackFunc)
	}
	return this
}
// OnError sets the callback invoked when request processing fails.
// Send still returns the error after this callback runs.
func (this *QRequest) OnError(callbackFunc func(error)) *QRequest {
	this.onErrorFunc = callbackFunc
	return this
}
func (this *QRequest) handleResponse(response *QResponse) (err error) {
	defer func() {
		if !response.bodyRead {
			closeErr := response.close()
			if err == nil {
				err = closeErr
			}
		}
	}()

	if callbackFunc := this.onStatusFunc[response.StatusCode()]; callbackFunc != nil {
		return callbackFunc(response)
	} else if this.onResponseFunc != nil {
		return this.onResponseFunc(response)
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

 if this.maxResponseSize < 0 {
  configErrors = append(configErrors, errors.New("riqu: maximum response size cannot be negative"))
 }

	if this.timeout < 0 {
		configErrors = append(configErrors, errors.New("riqu: timeout cannot be negative"))
	}
	if this.redirectOptions.Max < 0 {
		configErrors = append(configErrors, errors.New("riqu: redirect max cannot be negative"))
	}
	if this.redirectOptions.Responses > VAllResponses {
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
	requestContext := this.reqContext

 if requestContext == nil { requestContext = context.Background() }
 if err := requestContext.Err(); err != nil {
  this.handleResponseError(err)
  return err
 }

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

	var cookieChanges *QCookieJar

	if this.cookieJar != nil && this.cookieJar.jar != nil {
		httpClient.Jar = this.cookieJar.jar
		cookieChanges, err = NewCookieJar(&this.cookieJar.options)

		if err != nil {
			this.handleResponseError(err)
			return err
		}
	}
	if this.transport != nil && this.transport.transport != nil {
		httpClient.Transport = this.transport.transport
	}
	initialURL, err := url.Parse(this.url)

	if err != nil {
		this.handleResponseError(err)
		return err
	}
	currentURL := this.url

	if this.queryAll || len(this.query) > 0 {
		query := make(url.Values)
		if !this.queryAll {
			query, err = url.ParseQuery(initialURL.RawQuery)
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
		initialURL.RawQuery = query.Encode()
		initialURL.ForceQuery = false
		currentURL = initialURL.String()
	}
	currentMethod := this.method
	sendBody := currentMethod != "GET" && currentMethod != "HEAD" ||
		this.body.bodyType != vRequestBodyData || preparedBody.data != nil
	bodyDropped := false
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
			if preparedBody.reader != nil {
    bodyReader = io.NopCloser(preparedBody.reader.reader)
   } else if preparedBody.multipart != nil {
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
		httpReq, err := http.NewRequestWithContext(requestContext, currentMethod, currentURL, bodyReader)

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
			currentParsed, err := url.Parse(currentURL)

			if err != nil {
				this.handleResponseError(err)
				return err
			}
			sameOrigin := strings.EqualFold(initialURL.Scheme, currentParsed.Scheme) &&
				strings.EqualFold(initialURL.Host, currentParsed.Host)

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
					"Content-Encoding": true,
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
		if bodyDropped {
			httpReq.Header.Del("Content-Type")
			httpReq.Header.Del("Content-Encoding")
		} else if bodyReader != nil && (preparedBody.multipart != nil || (!this.header.Exist("Content-Type") && bodyContentType != "")) {
			httpReq.Header.Set("Content-Type", bodyContentType)
		}
		if cookieChanges != nil {
			replacedCookies := make(map[string]bool)

			for _, cookie := range cookieChanges.jar.Cookies(httpReq.URL) {
				replacedCookies[cookie.Name] = true
			}
			if len(replacedCookies) > 0 {
				cookies := httpReq.Cookies()
				httpReq.Header.Del("Cookie")

				for _, cookie := range cookies {
					if !replacedCookies[cookie.Name] {
						httpReq.AddCookie(cookie)
					}
				}
			}
		}
  if sendBody && preparedBody.reader != nil {
   if preparedBody.reader.isUsed {
    if httpReq.Body != nil { httpReq.Body.Close() }
    err := errors.New("riqu: body reader has already been used; set a new BodyReader before replaying")
    this.handleResponseError(err)
    return err
   }
   preparedBody.reader.isUsed = true
  }
  // RESPONSE
		httpRes, err := httpClient.Do(httpReq)

		if err != nil {
			this.handleResponseError(err)
			return err
		}
		if cookieChanges != nil {
			cookies := httpRes.Cookies()

			for _, cookie := range cookies {
				// Keep the scope even when the server deletes the cookie.
				cookie.Value = "changed"
				cookie.MaxAge = 0
				cookie.Expires = time.Time{}
				cookie.RawExpires = ""
			}
			cookieChanges.jar.SetCookies(httpReq.URL, cookies)
		}
  responseBody := httpRes.Body

  if this.maxResponseSize > 0 {
   responseBody = &tResponseLimitBody{ReadCloser: responseBody, remaining: this.maxResponseSize}
  }
  qRes := &QResponse{
			statusCode:    httpRes.StatusCode,
			status:        httpRes.Status,
			contentLength: httpRes.ContentLength,
			resHeader:     THeader(httpRes.Header).Clone(),
			resCookie:     httpRes.Cookies(),
			resURL:        currentURL,
			resMethod:     currentMethod,
			body:          responseBody,
		}

		if this.redirectOptions.Responses == VAllResponses {
			if err := this.handleResponse(qRes); err != nil {
				this.handleResponseError(err)
				return err
			}
		}
		follow := redirectCount < this.redirectOptions.Max &&
			isRedirectStatus(httpRes.StatusCode) &&
			httpRes.Header.Get("Location") != ""

		if !follow {
			if this.redirectOptions.Responses == VLastResponse {
				if err := this.handleResponse(qRes); err != nil {
					this.handleResponseError(err)
					return err
				}
			}
			return nil
		}
		if this.redirectOptions.Responses == VLastResponse {
			if err := qRes.close(); err != nil {
				this.handleResponseError(err)
				return err
			}
		}
		// REDIRECT - Set URL & Method
		redirectURL, err := url.Parse(httpRes.Header.Get("Location"))

		if err != nil {
			this.handleResponseError(err)
			return err
		}
		currentParsed, err := url.Parse(currentURL)

		if err != nil {
			this.handleResponseError(err)
			return err
		}
		currentURL = currentParsed.ResolveReference(redirectURL).String()

		switch httpRes.StatusCode {
		case 303:
			sendBody = false
			bodyDropped = true

			if currentMethod != "HEAD" {
				currentMethod = "GET"
			}
		case 301, 302:
			if currentMethod == "POST" {
				currentMethod = "GET"
				sendBody = false
				bodyDropped = true
			}
		}
		redirectCount++
	}
}
