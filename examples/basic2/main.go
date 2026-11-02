package main

import (
	"fmt"
	"github.com/rionvzxc/rxquest"
)

func main() {
	rxquest.New().
		GET("https://httpbin.org/get").
		OnSuccess(func(res *rxquest.QResponse) {
			text, _ := res.BodyText()

			fmt.Println("Status:", res.StatusCode)
			fmt.Println(text)
		}).
		OnError(func(err error) {
			fmt.Println(err)
		}).
		Send()
}