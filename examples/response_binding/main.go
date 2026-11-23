package main

import (
	"fmt"

	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		GET("https://httpbin.org/response-headers").
		SetQuery("X-Count", "12").
		AddQuery("Set-Cookie", "session=abc; Path=/; HttpOnly", "theme=dark; Path=/").
		OnResponse(func(res *riqu.QResponse) error {
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
			err := res.BindHeader(&headers)

			if err != nil {
				return err
			}
			var cookies struct {
				Session string `cookie:"session"`
				Theme   string `cookie:"theme"`
			}
			err = res.BindCookie(&cookies)

			if err != nil {
				return err
			}
			fmt.Printf("Bound headers: %+v\n", headers)
			fmt.Printf("Bound cookies: %+v\n", cookies)
			return nil
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}
