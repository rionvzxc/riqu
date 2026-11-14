package main

import (
	"fmt"
	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		POST("https://httpbin.org/post").
		Form(riqu.TValue{
			"name": "riqu",
			"version": "1.0",
		}).
		OnSuccess(func(res *riqu.QResponse) {
			text, err := res.Text()
			
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			fmt.Println(text)
		}).
		OnError(func(err error) {
			fmt.Println(err)
		}).
		Send()
}