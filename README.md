# riqu

`riqu` is a lightweight HTTP client library for Go, designed with a simple and chainable API. It makes sending HTTP requests, handling responses, managing cookies, and downloading files straightforward and convenient.

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
- Success and error callbacks
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
        OnSuccess(func(res *riqu.QResponse) {
            text, err := res.Text()
            
            if err != nil {
                fmt.Println("Error:", err)
                return
            }
            fmt.Println(res.StatusCode)
            fmt.Println(text)
        }).
        OnError(func(err error) {
            fmt.Println(err)
        }).
        Send()
}
```

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
    OnSuccess(func(res *riqu.QResponse) {
        fmt.Println("Status:", res.StatusCode)
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

To run an example, clone the repository and execute:

```bash
go run ./examples/basic
```

## License

See [LICENSE](LICENSE) for details.
