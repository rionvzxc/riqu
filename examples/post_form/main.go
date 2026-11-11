package main

import (
	"fmt"
	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		POST("https://httpbin.org/post").
		SetForm(riqu.TValue{
			"name": "riqu",
			"version": "1.0",
		}).
		OnSuccess(func(res *riqu.QResponse) {
			text := res.GetText()
			fmt.Println(text)
		}).
		OnError(func(err error) {
			fmt.Println(err)
		}).
		Send()
}