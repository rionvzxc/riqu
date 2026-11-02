package main

import (
	"fmt"
	"github.com/rionvzxc/rxquest"
)

func main() {
	jar := rxquest.NewCookieJar(nil)
	client := rxquest.New().CookieJar(jar)

	// Set Cookie
	client.GET("https://httpbin.org/cookies/set?session_id=abc123456&username=rxquest").
		OnSuccess(func(res *rxquest.QResponse) {
			body, _ := res.BodyText()

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
	cookies := jar.Cookies("https://httpbin.org")

	for _, cookie := range cookies {
		fmt.Printf("%s = %s\n", cookie.Name, cookie.Value)
	}
}