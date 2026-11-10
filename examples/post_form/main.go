package main

import (
	"fmt"
	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		POST("https://httpbin.org/post").
		BodyForm(riqu.TValue{
			"name": "riqu",
			"version": "1.0",
		}).
		OnSuccess(func(res *riqu.QResponse) {
			text, _ := res.BodyText()
			fmt.Println(text)
		}).
		OnError(func(err error) {
			fmt.Println(err)
		}).
		Send()
}