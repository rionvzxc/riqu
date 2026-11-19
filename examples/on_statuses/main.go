package main

import (
	"fmt"
	"net/http"

	"github.com/rionvzxc/riqu"
)

func main() {
	handleUnavailable := func(res *riqu.QResponse) {
		fmt.Println("Service unavailable, try again later. Status:", res.StatusCode)
	}

	riqu.New().
		GET("https://httpbin.org/status/503").
		OnStatuses([]int{
			http.StatusTooManyRequests,
			http.StatusServiceUnavailable,
		}, handleUnavailable).
		OnSuccess(func(res *riqu.QResponse) {
			fmt.Println("Status:", res.StatusCode)
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}
