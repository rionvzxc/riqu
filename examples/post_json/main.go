package main

import (
	"fmt"
	"github.com/rionvzxc/rxquest"
)

func main() {
	rxquest.New().
		POST("https://httpbin.org/post").
		BodyJSON(rxquest.TMap{
			"name":    "rxquest",
			"version": "1.0",
		}).
		OnSuccess(func(res *rxquest.QResponse) {
			var data struct {
				JSON rxquest.TMap `json:"json"`
			}
			if err := res.BodyJSON(&data); err != nil {
				fmt.Println("JSON Error:", err)
				return
			}
			fmt.Println("Status:", res.StatusCode)
			fmt.Println("Response JSON:")

			for key, value := range data.JSON {
				fmt.Println(key, "=", value)
			}
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}