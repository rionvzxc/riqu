package main

import (
	"fmt"

	"github.com/rionvzxc/riqu"
)

func main() {
	cookieJar, err := riqu.NewCookieJar(nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

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
	cookies, err := cookieJar.Cookies("https://httpbin.org")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, cookie := range cookies {
		fmt.Printf("%s = %s\n", cookie.Name, cookie.Value)
	}
}