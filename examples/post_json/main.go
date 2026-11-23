package main

import (
	"fmt"
	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		POST("https://httpbin.org/post").
		JSON(riqu.TMap{
			"name":    "riqu",
			"version": "1.0",
			"config": riqu.TMap{
				"timeout": 30,
				"retry":   3,
			},
		}).
		OnResponse(func(res *riqu.QResponse) error {
			var data struct {
				JSON riqu.TMap `json:"json"`
			}
			if err := res.BindJSON(&data); err != nil {
				return err
			}
			fmt.Println("Status:", res.StatusCode())
			fmt.Println("Response JSON:")

			for key, value := range data.JSON {
				fmt.Printf("%s = %v\n", key, value)

				if nested, ok := value.(riqu.TMap); ok {
					for nestedKey, nestedValue := range nested {
						fmt.Printf("  %s = %v\n", nestedKey, nestedValue)
					}
				}
			}
			return nil
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}