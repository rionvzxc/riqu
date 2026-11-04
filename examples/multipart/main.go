package main

import (
	"fmt"

	"github.com/rionvzxc/rxquest"
)

func main() {
	rxquest.New().
		POST("https://httpbin.org/post").
		BodyMultipart(
			rxquest.TValue{
				"name":    "rxquest",
				"version": "1.0",
				"type":    "multipart",
			},
			rxquest.TValue{
				"samplefile": "examples/multipart/sample.txt",
			},
		).
		OnSuccess(func(res *rxquest.QResponse) {
			text, _ := res.BodyText()

			fmt.Println("Status:", res.StatusCode)
			fmt.Println(text)
		}).
		OnError(func(err error) {
			fmt.Println(err)
		}).
		Send()
}