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
		OnSuccess(func(res *riqu.QResponse) {
			fmt.Println("Success:", res.StatusCode)
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}