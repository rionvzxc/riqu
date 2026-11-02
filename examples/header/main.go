package main

import (
	"fmt"
	"github.com/rionvzxc/rxquest"
)

func main() {
	rxquest.New().
		GET("https://httpbin.org/headers").
		Header(rxquest.THeader{
			"X-App":     {"rxquest"},
			"X-Version": {"1.0"},
			"User-Agent": {"RxquestClient/1.0"},
			"Accept":     {"application/json"},
		}).
		SetHeader("X-Test", "value1", "value2", "value3").
		OnSuccess(func(res *rxquest.QResponse) {
			text, _ := res.BodyText()
			fmt.Println(text)
		}).
		OnError(func(err error) {
			fmt.Println(err)
		}).
		Send()
}