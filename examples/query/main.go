package main

import (
	"fmt"
	"log"

	"github.com/rionvzxc/riqu"
)

func main() {
	var responseErr error
	err := riqu.New().
		GET("https://httpbin.org/get?old=removed").
		SetAllQueries(riqu.TValues{
			"q":    {"initial"},
			"page": {"1"},
			"tag":  {"go"},
		}).
		SetQuery("q", "go http").
		AddQuery("tag", "client", "api").
		OnSuccess(func(res *riqu.QResponse) {
			fmt.Println("Request URL:", res.Url())
			fmt.Println("Query:", res.Query("q", ""))
			fmt.Println("Tags:", res.QueryValues("tag"))
			fmt.Println("All queries:", res.AllQueries())

			var query struct {
				Search string   `query:"q"`
				Page   int      `query:"page"`
				Tags   []string `query:"tag"`
			}
			responseErr = res.BindQuery(&query)

			if responseErr != nil {
				return
			}
			fmt.Printf("Bound query: %+v\n", query)
		}).
		Send()
	if err != nil {
		log.Fatal(err)
	}
	if responseErr != nil {
		log.Fatal(responseErr)
	}
}
