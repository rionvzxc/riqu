package main

import (
	"fmt"

	"github.com/rionvzxc/riqu"
)

func main() {
	// This endpoint returns 1024 bytes, intentionally exceeding the 512-byte limit.
	riqu.New().
		GET("https://httpbin.org/bytes/1024").
		MaxResponseSize(512).
		OnResponse(func(res *riqu.QResponse) error {
			_, err := res.Data()
			return err
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}
