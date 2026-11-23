package main

import (
	"fmt"
	"log"

	"github.com/rionvzxc/riqu"
)

func main() {
	var responseErr error
	err := riqu.New().
		GET("https://httpbin.org/response-headers").
		SetQuery("X-Count", "12").
		AddQuery("Set-Cookie", "session=abc; Path=/; HttpOnly", "theme=dark; Path=/").
		OnSuccess(func(res *riqu.QResponse) {
			fmt.Println("Count header:", res.Header("X-Count", "0"))
			fmt.Println("Set-Cookie headers:", res.HeaderValues("Set-Cookie"))
			fmt.Println("All headers:", res.AllHeaders())
			fmt.Println("All cookies:", res.AllCookies())

			if cookie := res.Cookie("session"); cookie != nil {
				fmt.Println("Session:", cookie.Value, "HttpOnly:", cookie.HttpOnly)
			}
			fmt.Println("Session cookies:", res.Cookies("session"))

			var headers struct {
				Count int `header:"X-Count"`
			}
			responseErr = res.BindHeader(&headers)

			if responseErr != nil {
				return
			}
			var cookies struct {
				Session string `cookie:"session"`
				Theme   string `cookie:"theme"`
			}
			responseErr = res.BindCookie(&cookies)

			if responseErr != nil {
				return
			}
			fmt.Printf("Bound headers: %+v\n", headers)
			fmt.Printf("Bound cookies: %+v\n", cookies)
		}).
		Send()
	if err != nil {
		log.Fatal(err)
	}
	if responseErr != nil {
		log.Fatal(responseErr)
	}
}
