package main

import (
	"fmt"
	"github.com/rionvzxc/rxquest"
)

func main() {
	rxquest.New().
		POST("https://httpbin.org/post").
		BodyForm(rxquest.TValue{
			"name": "rxquest",
			"version": "1.0",
		}).
		OnSuccess(func(res *rxquest.QResponse) {
			text, _ := res.BodyText()
			fmt.Println(text)
		}).
		OnError(func(err error) {
			fmt.Println(err)
		}).
		Send()
}