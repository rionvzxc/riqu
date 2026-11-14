package main

import (
	"fmt"

	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		POST("https://httpbin.org/post").
		FormMultipart(riqu.TMultipart{
			Form: riqu.TValues{
				"name":    {"riqu"},
				"version": {"1.0"},
				"type":    {"multipart"},
			},
			File: riqu.TValues{
				"samplefile": {"examples/multipart/sample.txt"},
			},
		}).
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
