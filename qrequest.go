package riqu

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type QRequest struct {
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
	isAllQueriesSet  bool
	transport *QTransport

	onStatusFunc  map[int]func(*QResponse) error
	onResponseFunc func(*QResponse) error
	onErrorFunc   func(error)
	maxResponseSize int64
	uploadOptions tTransferOptions
	downloadOptions tTransferOptions
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
// The parent context can still cancel the request or impose an earlier deadline.
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
		if !response.isBodyRead {
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
	// CONFIGURATION - Validate the request settings before preparing the body.
	var configErrors []error

	if this.maxResponseSize < 0 {
		configErrors = append(configErrors, errors.New("riqu: maximum response size cannot be negative"))
	}
	if this.uploadOptions.interval < 0 || this.downloadOptions.interval < 0 {
		configErrors = append(configErrors, errors.New("riqu: progress interval cannot be negative"))
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
	// BODY - Encode the configured value or prepare the multipart description.
	preparedBody, err := this.prepareBody()

	if err != nil {
		this.handleResponseError(err)
		return err
	}
	// CONTEXT - Share cancellation and timeout across requests and body reads.
	reqContext := this.reqContext

	if reqContext == nil {
		reqContext = context.Background()
	}
	if err := reqContext.Err(); err != nil {
		this.handleResponseError(err)
		return err
	}
	if this.timeout > 0 {
		var cancel context.CancelFunc
		reqContext, cancel = context.WithTimeout(reqContext, this.timeout)
		defer cancel()
	}
	// CLIENT - Handle redirects here and reuse the configured jar and transport.
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// net/http may prepare an unused replay body before checking redirects.
			if req.Body != nil {
				req.Body.Close()
			}
			return http.ErrUseLastResponse
		},
	}
	// COOKIES - Track changed scopes without modifying the manual cookies.
	var cookieChanges *QCookieJar

	if this.cookieJar != nil && this.cookieJar.jar != nil {
		httpClient.Jar = this.cookieJar.jar
		cookieChanges, err = NewCookieJar(&this.cookieJar.options)

		if err != nil {
			this.handleResponseError(err)
			return err
		}
	}
	if this.transport != nil && this.transport.httpTransport != nil {
		httpClient.Transport = this.transport.httpTransport
	}
	// URL - Apply configured query values only to the initial request.
	initialURL, err := url.Parse(this.url)

	if err != nil {
		this.handleResponseError(err)
		return err
	}
	currURL := this.url

	if this.isAllQueriesSet || len(this.query) > 0 {
		query := make(url.Values)

		if !this.isAllQueriesSet {
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
		currURL = initialURL.String()
	}
	// REDIRECT STATE - Keep the original headers and track method/body changes.
	currMethod := this.method
	isBodyEnabled := currMethod != "GET" && currMethod != "HEAD" ||
		this.body.bodyType != vRequestBodyData || preparedBody.data != nil
	isBodyDropped := false
	redirectCount := 0
	initialRequest := &http.Request{Header: http.Header(this.header.Clone())}

	for _, cookie := range this.cookies {
		initialRequest.AddCookie(cookie)
	}
	for {
		// BODY READER - Create a fresh reader for each request, including replays.
		var bodyReader io.Reader
		var bodyCloser io.Closer
		bodyContentType := preparedBody.contentType

		if isBodyEnabled {
			if preparedBody.reader != nil {
				// The transport must not close a reader owned by the caller.
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
		// REQUEST - Attach the shared context to the current URL, method, and body.
		httpReq, err := http.NewRequestWithContext(reqContext, currMethod, currURL, bodyReader)

		if err != nil {
			if bodyCloser != nil {
				bodyCloser.Close()
			}
			this.handleResponseError(err)
			return err
		}
		// HEADERS - Copy initial headers or filter them for a different origin.
		if redirectCount == 0 {
			httpReq.Header = initialRequest.Header.Clone()
		} else {
			currParsed, err := url.Parse(currURL)

			if err != nil {
				this.handleResponseError(err)
				return err
			}
			isSameOrigin := strings.EqualFold(initialURL.Scheme, currParsed.Scheme) &&
				strings.EqualFold(initialURL.Host, currParsed.Host)

			if isSameOrigin {
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
		// HOST - Move the Host override into the field used by net/http.
		if host := httpReq.Header.Get("Host"); host != "" {
			httpReq.Host = host
		}
		httpReq.Header.Del("Host")
		// BODY HEADERS - Update the content type or remove metadata for a dropped body.
		_, isContentTypeSet := httpReq.Header["Content-Type"]

		if isBodyDropped {
			httpReq.Header.Del("Content-Type")
			httpReq.Header.Del("Content-Encoding")
		} else if bodyReader != nil && (preparedBody.multipart != nil || (!isContentTypeSet && bodyContentType != "")) {
			httpReq.Header.Set("Content-Type", bodyContentType)
		}
		// REQUEST COOKIES - Remove manual values replaced within this URL's scope.
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
		if isBodyEnabled && preparedBody.reader != nil {
			if preparedBody.reader.isUsed {
				if httpReq.Body != nil {
					httpReq.Body.Close()
				}
				err := errors.New("riqu: body reader has already been used; set a new BodyReader before replaying")
				this.handleResponseError(err)
				return err
			}
			preparedBody.reader.isUsed = true
		}
		// UPLOAD - Observe transport reads only when transfer callbacks are configured.
		var uploadBody *tTransferBody
		var uploadBodiesMutex sync.Mutex

		if this.uploadOptions.enabled() && httpReq.Body != nil && httpReq.Body != http.NoBody {
			total := httpReq.ContentLength

			if total == 0 {
				total = -1
			}
			uploadBody = &tTransferBody{
				body: httpReq.Body,
				options: this.uploadOptions,
				info: TTransferInfo{
					URL: currURL,
					Method: currMethod,
					Total: total,
				},
			}
			httpReq.Body = uploadBody
			// Keep native retries replayable and observe each body that is actually read.
			if getBody := httpReq.GetBody; getBody != nil {
				uploadOptions := this.uploadOptions
				transferInfo := uploadBody.info
				httpReq.GetBody = func() (io.ReadCloser, error) {
					replayReader, replayErr := getBody()

					if replayErr != nil {
						if replayReader != nil { replayReader.Close() }
						return nil, replayErr
					}
					if replayReader == nil {
						return nil, errors.New("riqu: GetBody returned a nil reader")
					}
					uploadBodiesMutex.Lock()
					defer uploadBodiesMutex.Unlock()
					uploadBody = &tTransferBody{
						body: replayReader,
						options: uploadOptions,
						info: transferInfo,
						prevBody: uploadBody,
					}
					return uploadBody, nil
				}
			}
		}
		// SEND - Execute the request using the shared context and configured transport.
		httpRes, err := httpClient.Do(httpReq)

		// UPLOAD END - Release the body before reporting the completed attempt.
		uploadBodiesMutex.Lock()
		lastUploadBody := uploadBody
		uploadBodiesMutex.Unlock()

		if lastUploadBody != nil {
			func() {
				isCompleted := false
				defer func() {
					if httpRes != nil && (!isCompleted || err != nil) {
						httpRes.Body.Close()
					}
				}()

				for body := lastUploadBody; body != nil; body = body.prevBody {
					if closeErr := body.Close(); closeErr != nil {
						err = errors.Join(err, closeErr)
					}
					body.finish(err, true)
				}
				isCompleted = true
			}()
		}
		if err != nil {
			this.handleResponseError(err)
			return err
		}
		// RESPONSE COOKIES - Remember changed scopes, including cookie deletions.
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
		// RESPONSE BODY - Bound decoded reads before observing consumed byte counts.
		responseBody := httpRes.Body

		if this.maxResponseSize > 0 {
			responseBody = &tResponseLimitBody{ReadCloser: responseBody, remaining: this.maxResponseSize}
		}
		var downloadBody *tTransferBody

		if this.downloadOptions.enabled() {
			downloadBody = &tTransferBody{
				body: responseBody,
				options: this.downloadOptions,
				info: TTransferInfo{
					URL: currURL,
					Method: currMethod,
					Total: httpRes.ContentLength,
				},
			}
			responseBody = downloadBody
		}
		// RESPONSE - Capture metadata and transfer body handling to QResponse.
		response := &QResponse{
			statusCode:    httpRes.StatusCode,
			status:        httpRes.Status,
			contentLength: httpRes.ContentLength,
			resHeader:     THeader(httpRes.Header).Clone(),
			resCookie:     httpRes.Cookies(),
			resURL:        currURL,
			resMethod:     currMethod,
			body:          responseBody,
			transferBody:  downloadBody,
		}
		// CALLBACKS - Deliver each response or only the final response as configured.
		if this.redirectOptions.Responses == VAllResponses {
			if err := this.handleResponse(response); err != nil {
				this.handleResponseError(err)
				return err
			}
		}
		isRedirectAllowed := redirectCount < this.redirectOptions.Max &&
			isRedirectStatus(httpRes.StatusCode) &&
			httpRes.Header.Get("Location") != ""

		if !isRedirectAllowed {
			if this.redirectOptions.Responses == VLastResponse {
				if err := this.handleResponse(response); err != nil {
					this.handleResponseError(err)
					return err
				}
			}
			return nil
		}
		// CLEANUP - Close an intermediate body that was not delivered to a callback.
		if this.redirectOptions.Responses == VLastResponse {
			if err := response.close(); err != nil {
				this.handleResponseError(err)
				return err
			}
		}
		// REDIRECT URL - Resolve Location relative to the current URL.
		redirectURL, err := url.Parse(httpRes.Header.Get("Location"))

		if err != nil {
			this.handleResponseError(err)
			return err
		}
		currParsed, err := url.Parse(currURL)

		if err != nil {
			this.handleResponseError(err)
			return err
		}
		currURL = currParsed.ResolveReference(redirectURL).String()
		// REDIRECT BODY - Drop or replay the body according to the status and method.
		switch httpRes.StatusCode {
		case 303:
			isBodyEnabled = false
			isBodyDropped = true

			if currMethod != "HEAD" {
				currMethod = "GET"
			}
		case 301, 302:
			if currMethod == "POST" {
				currMethod = "GET"
				isBodyEnabled = false
				isBodyDropped = true
			}
		}
		redirectCount++
	}
}
