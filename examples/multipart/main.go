package main

import (
	"fmt"

	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		POST("https://httpbin.org/post").
		SetMultipart(
			riqu.TValue{
				"name":    "riqu",
				"version": "1.0",
				"type":    "multipart",
			},
			riqu.TValue{
				"samplefile": "examples/multipart/sample.txt",
			},
		).
		OnSuccess(func(res *riqu.QResponse) {
			text := res.GetText()

			fmt.Println("Status:", res.StatusCode)
			fmt.Println(text)
		}).
		OnError(func(err error) {
			fmt.Println(err)
		}).
		Send()
}