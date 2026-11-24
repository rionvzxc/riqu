package main

import (
	"fmt"

	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		GET("https://httpbin.org/get?old=removed").
		SetAllQueries(riqu.TValues{
			"q":    {"initial"},
			"page": {"1"},
			"tag":  {"go"},
		}).
		SetQuery("q", "go http").
		AddQuery("tag", "client", "api").
		OnResponse(func(res *riqu.QResponse) error {
			fmt.Println("Request URL:", res.URL())
			fmt.Println("Query:", res.Query("q", ""))
			fmt.Println("Tags:", res.QueryValues("tag"))
			fmt.Println("All queries:", res.AllQueries())

			var query struct {
				Search string   `query:"q"`
				Page   int      `query:"page"`
				Tags   []string `query:"tag"`
			}
			err := res.BindQuery(&query)

			if err != nil {
				return err
			}
			fmt.Printf("Bound query: %+v\n", query)
			return nil
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}
