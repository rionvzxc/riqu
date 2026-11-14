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
- Raw, Text, HTML, JSON, Form, and Multipart request bodies
- Response handling: Raw, Text, JSON, and File
- Cookie and Cookie Jar management
- Automatic redirect handling
- Request timeout configuration
- Success and error callbacks

## Request Bodies

`Text()` sets `Content-Type` to `text/plain`, `HTML()` sets it to `text/html`, `JSON()` sets it to `application/json`, and `Form()` sets it to `application/x-www-form-urlencoded`. An explicit `Content-Type` set with `Header()` takes precedence over these defaults. JSON, text, and form bodies are encoded when `Send()` is called.

`FormMultipart()` streams file contents into the request instead of buffering the complete multipart body in memory. The request may use chunked transfer encoding when its content length is unknown. Multipart files are reopened if a redirect requires the body to be sent again.

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

To run an example, clone the repository and execute:

```bash
go run ./examples/basic
```

## Redirects

Redirects are followed automatically by default, with a maximum of 10 redirects. The final response is returned.

## License

See [LICENSE](LICENSE) for details.
