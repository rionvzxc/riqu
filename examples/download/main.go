package main

import (
	"fmt"
	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		GET("https://httpbin.org/image/png").
		OnResponse(func(res *riqu.QResponse) error {
			err := res.File(&riqu.TFileOptions{
				Dir: "downloads",
				DefaultFilename: "httpbin-image",
			})
			if err != nil {
				return err
			}
			fmt.Println("Download Success!")
			fmt.Println("Status:", res.StatusCode())
			fmt.Println("Content-Type:", res.Header("Content-Type", ""))
			return nil
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}