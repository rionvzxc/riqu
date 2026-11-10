package main

import (
	"fmt"
	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		Url("https://httpbin.org/get").
		Method("GET").
		OnSuccess(func(res *riqu.QResponse) {
			text, _ := res.BodyText()

			fmt.Println("Status:", res.StatusCode)
			fmt.Println(text)
		}).
		OnError(func(err error) {
			fmt.Println(err)
		}).
		Send()
}