package main

import (
	"fmt"
	"github.com/rionvzxc/rxquest"
)

func main() {
	rxquest.New().
		GET("https://httpbin.org/image/png").
		OnSuccess(func(res *rxquest.QResponse) {
			err := res.BodyFile(&rxquest.TFileOptions{
				Dir: "downloads",
				DefaultFilename: "httpbin-image",
			})
			if err != nil {
				fmt.Println("Download Error:", err)
				return
			}
			fmt.Println("Download Success!")
			fmt.Println("Status:", res.StatusCode)
			fmt.Println("Content-Type:", res.Header.Get("Content-Type"))
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}