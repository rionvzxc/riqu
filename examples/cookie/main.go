package main

import (
	"fmt"
	"github.com/rionvzxc/riqu"
)

func main() {
	client := riqu.New().CookieJar(nil)

	// Set Cookie
	client.GET("https://httpbin.org/cookies/set?session_id=abc123456&username=riqu").
		OnSuccess(func(res *riqu.QResponse) {
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
	jar := client.GetCookieJar()
	cookies := jar.GetCookies("https://httpbin.org")

	for _, cookie := range cookies {
		fmt.Printf("%s = %s\n", cookie.Name, cookie.Value)
	}
}