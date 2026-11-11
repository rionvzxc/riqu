package main

import (
	"fmt"
	"github.com/rionvzxc/riqu"
)

func main() {
	cookieJar := riqu.NewCookieJar(nil)

	// Set Cookie
	riqu.New().
		CookieJar(cookieJar).
		GET("https://httpbin.org/cookies/set?session_id=abc123456&username=riqu").
		OnSuccess(func(res *riqu.QResponse) {
			body := res.Text()

			fmt.Println("Response:")
			fmt.Println("Status:", res.StatusCode)
			fmt.Println(body)
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()

	// Get Cookie from Jar
	fmt.Println("\nCookie Jar:")
	cookies := cookieJar.GetCookies("https://httpbin.org")

	for _, cookie := range cookies {
		fmt.Printf("%s = %s\n", cookie.Name, cookie.Value)
	}
}