# riqu

`riqu` is an HTTP client library for Go with a simple, chainable API for building requests and handling responses. It supports query parameters, custom headers, JSON and form data, multipart uploads, cookie management, redirects, and file downloads. Status-specific callbacks and reusable transports help you configure response handling, proxies, and trusted CA certificates.

## Installation

```bash
go get github.com/rionvzxc/riqu
```

## Features

- Fluent and chainable API
- HTTP request methods: GET, POST, PUT, DELETE, PATCH
- Custom HTTP methods
- Custom request headers
- Query parameter replacement and appending
- Raw, Text, HTML, JSON, Form, and Multipart request bodies
- Response handling: Raw, Text, JSON, and File
- Cookie and Cookie Jar management
- Automatic redirect handling
- Request timeout configuration
- Response and error callbacks
- Status-specific response callbacks
- Reusable transports with proxy and additional CA configuration

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/rionvzxc/riqu"
)

func main() {
    riqu.New().
        GET("https://httpbin.org/get").
        OnResponse(func(res *riqu.QResponse) error {
            text, err := res.Text()
            
            if err != nil {
                return err
            }
            fmt.Println(res.StatusCode())
            fmt.Println(text)
            return nil
        }).
        OnError(func(err error) {
            fmt.Println(err)
        }).
        Send()
}
```

## Error Handling

`OnError` handles errors from request preparation, sending the request, and response processing, including errors returned by `OnResponse`, `OnStatus`, and `OnStatuses` callbacks. Return an error from a response callback to pass it to `OnError`, or return `nil` when processing succeeds or the error has already been handled. `Send()` still returns the error after `OnError` runs. HTTP 4xx and 5xx responses do not automatically become errors.

## Transport

riqu supports reusable transports for HTTP and HTTPS proxies, additional trusted
CA certificates, and connection reuse across requests. TLS certificate and
hostname verification remain enabled. Empty options use the environment proxy
configuration and system CA certificates.

```go
transport, err := riqu.NewTransport(&riqu.TTransportOptions{
    ProxyURL: "http://127.0.0.1:8080",
})
if err != nil {
    log.Fatal(err)
}
defer transport.CloseIdleConnections()

err = riqu.New().
    GET("https://httpbin.org/get").
    Transport(transport).
    OnResponse(func(res *riqu.QResponse) error {
        fmt.Println("Status:", res.StatusCode())
        return nil
    }).
    Send()
if err != nil {
    log.Fatal(err)
}
```

## Examples

Runnable examples are available in the repository:

| Example | Description |
|---|---|
| [basic](https://github.com/rionvzxc/riqu/tree/main/examples/basic) | Basic HTTP request |
| [basic2](https://github.com/rionvzxc/riqu/tree/main/examples/basic2) | Basic request example |
| [header](https://github.com/rionvzxc/riqu/tree/main/examples/header) | Custom request headers |
| [query](https://github.com/rionvzxc/riqu/tree/main/examples/query) | Query setters, response query access, and binding |
| [response_binding](https://github.com/rionvzxc/riqu/tree/main/examples/response_binding) | Response header and cookie access and binding |
| [cookie](https://github.com/rionvzxc/riqu/tree/main/examples/cookie) | Cookie Jar management |
| [post_json](https://github.com/rionvzxc/riqu/tree/main/examples/post_json) | JSON request and response |
| [post_form](https://github.com/rionvzxc/riqu/tree/main/examples/post_form) | Form URL-encoded request |
| [multipart](https://github.com/rionvzxc/riqu/tree/main/examples/multipart) | Multipart form and file upload |
| [download](https://github.com/rionvzxc/riqu/tree/main/examples/download) | Download files |
| [redirect](https://github.com/rionvzxc/riqu/tree/main/examples/redirect) | Follow an HTTP redirect |
| [timeout](https://github.com/rionvzxc/riqu/tree/main/examples/timeout) | Request timeout and error handling |
| [on_status](https://github.com/rionvzxc/riqu/tree/main/examples/on_status) | Handle a specific HTTP status |
| [on_statuses](https://github.com/rionvzxc/riqu/tree/main/examples/on_statuses) | Handle multiple HTTP statuses |
| [transport](https://github.com/rionvzxc/riqu/tree/main/examples/transport) | Proxy and additional CA configuration |
| [context](https://github.com/rionvzxc/riqu/tree/main/examples/context) | Cancellation and deadlines |
| [write_to](https://github.com/rionvzxc/riqu/tree/main/examples/write_to) | Stream a response body to a writer |
| [body_reader](https://github.com/rionvzxc/riqu/tree/main/examples/body_reader) | Send a one-shot body from a reader |
| [response_limit](https://github.com/rionvzxc/riqu/tree/main/examples/response_limit) | Limit response body size |

To run an example, clone the repository and execute:

```bash
go run ./examples/basic
```

## License

See [LICENSE](LICENSE) for details.
