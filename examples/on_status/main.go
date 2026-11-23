package main

import (
	"fmt"
	"net/http"

	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		GET("https://httpbin.org/status/404").
		OnStatus(http.StatusNotFound, func(res *riqu.QResponse) {
			fmt.Println("Resource not found:", res.Url())
		}).
		OnSuccess(func(res *riqu.QResponse) {
			fmt.Println("Status:", res.StatusCode())
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}
