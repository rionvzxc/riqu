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
- Raw, Text, JSON, Form, and Multipart request bodies
- Response handling: Raw, Text, JSON, and File
- Cookie and Cookie Jar management
- Automatic redirect handling
- Request timeout configuration
- Success and error callbacks

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
			body, _ := res.BodyText()
			fmt.Println(res.StatusCode)
			fmt.Println(body)
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
| [timeout](https://github.com/rionvzxc/riqu/tree/main/examples/timeout) | Request timeout and error handling |

To run an example, clone the repository and execute:

```bash
go run ./examples/basic
```

## Redirects

Redirects are followed automatically by default, with a maximum of 10 redirects. The final response is returned.

## License

See [LICENSE](LICENSE) for details.