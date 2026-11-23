package main

import (
	"fmt"
	"time"
	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		Url("https://httpbin.org/delay/5").
		Timeout(2 * time.Second).
		OnResponse(func(res *riqu.QResponse) error {
			fmt.Println("Success:", res.StatusCode())
			return nil
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}