package main

import (
	"fmt"
	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		GET("https://httpbin.org/get").
		OnSuccess(func(res *riqu.QResponse) {
			text, err := res.Text()
			
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			fmt.Println("Status:", res.StatusCode)
			fmt.Println(text)
		}).
		OnError(func(err error) {
			fmt.Println(err)
		}).
		Send()
}