package main

import (
	"fmt"
	"net/http"

	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		GET("https://httpbin.org/status/404").
		OnStatus(http.StatusNotFound, func(res *riqu.QResponse) error {
			fmt.Println("Resource not found:", res.URL())
			return nil
		}).
		OnResponse(func(res *riqu.QResponse) error {
			fmt.Println("Status:", res.StatusCode())
			return nil
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}
