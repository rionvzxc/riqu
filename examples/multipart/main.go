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
		OnResponse(func(res *riqu.QResponse) error {
			text, err := res.Text()

			if err != nil {
				return err
			}
			fmt.Println("Status:", res.StatusCode())
			fmt.Println(text)
			return nil
		}).
		OnError(func(err error) {
			fmt.Println(err)
		}).
		Send()
}
