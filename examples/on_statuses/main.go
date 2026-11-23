package main

import (
	"fmt"
	"net/http"

	"github.com/rionvzxc/riqu"
)

func main() {
	handleUnavailable := func(res *riqu.QResponse) error {
		fmt.Println("Service unavailable, try again later. Status:", res.StatusCode())
		return nil
	}

	riqu.New().
		GET("https://httpbin.org/status/503").
		OnStatuses([]int{
			http.StatusTooManyRequests,
			http.StatusServiceUnavailable,
		}, handleUnavailable).
		OnResponse(func(res *riqu.QResponse) error {
			fmt.Println("Status:", res.StatusCode())
			return nil
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}
