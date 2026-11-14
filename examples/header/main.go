package main

import (
	"fmt"
	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		GET("https://httpbin.org/headers").
		Header(riqu.THeader{
			"X-App":     {"riqu"},
			"X-Version": {"1.0"},
			"User-Agent": {"riquClient/1.0"},
			"Accept":     {"application/json"},
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