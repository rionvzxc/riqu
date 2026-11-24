package main

import (
	"context"
	"fmt"
	"time"

	"github.com/rionvzxc/riqu"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	riqu.New().GET("https://httpbin.org/get").Context(ctx).
		OnResponse(func(res *riqu.QResponse) error {
			text, err := res.Text()
			if err != nil {
				return err
			}
			fmt.Println(text)
			return nil
		}).OnError(func(err error) {
			fmt.Println("Error:", err)
		}).Send()
}
