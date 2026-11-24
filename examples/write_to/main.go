package main

import (
	"fmt"
	"os"

	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		GET("https://httpbin.org/get").
		OnResponse(func(res *riqu.QResponse) error {
			written, err := res.WriteTo(os.Stdout)

			if err != nil {
				return err
			}
			fmt.Printf("\nBytes written: %d\n", written)
			return nil
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}
