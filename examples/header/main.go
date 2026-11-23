package main

import (
	"fmt"
	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		GET("https://httpbin.org/headers").
		SetAllHeaders(riqu.THeader{
			"X-App":     {"riqu"},
			"X-Version": {"1.0"},
			"User-Agent": {"riquClient/1.0"},
			"Accept":     {"application/json"},
		}).
		SetHeader("X-Version", "2.0").
		AddHeader("X-Tag", "go", "client").
		OnResponse(func(res *riqu.QResponse) error {
			text, err := res.Text()

			if err != nil {
				return err
			}
			fmt.Println(text)
			return nil
		}).
		OnError(func(err error) {
			fmt.Println(err)
		}).
		Send()
}
