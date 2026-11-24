package main

import (
	"fmt"
	"strings"

	"github.com/rionvzxc/riqu"
)

func main() {
	reader := strings.NewReader("Hello from riqu!")

	riqu.New().
		POST("https://httpbin.org/post").
		BodyReader(reader).
		ContentType("text/plain").
		OnResponse(func(res *riqu.QResponse) error {
			text, err := res.Text()

			if err != nil {
				return err
			}
			fmt.Println(text)
			return nil
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}
