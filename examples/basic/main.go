package main

import (
	"fmt"
	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		URL("https://httpbin.org/get").
		Method("GET").
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