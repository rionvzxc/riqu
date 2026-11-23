package main

import (
	"fmt"

	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		GET("https://httpbin.org/redirect/1").
		OnSuccess(func(res *riqu.QResponse) {
			text, err := res.Text()

			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			fmt.Println("Status:", res.StatusCode())
			fmt.Println("Request URL:", res.Url())
			fmt.Println("Request Method:", res.Method())
			fmt.Println(text)
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}
