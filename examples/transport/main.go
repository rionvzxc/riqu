package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/rionvzxc/riqu"
)

func main() {
	targetURL := flag.String("url", "https://httpbin.org/get", "URL to request")
	proxyURL := flag.String("proxy", "", "HTTP or HTTPS proxy URL")
	caFile := flag.String("ca", "", "Additional CA certificates in PEM format")
	flag.Parse()

	transport, err := riqu.NewTransport(&riqu.TTransportOptions{
		ProxyURL: *proxyURL,
		CAFile:   *caFile,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer transport.CloseIdleConnections()

	riqu.New().GET(*targetURL).
		Transport(transport).
		OnResponse(func(res *riqu.QResponse) error {
			text, err := res.Text()

			if err != nil {
				return err
			}
			fmt.Println("Status:", res.StatusCode())
			fmt.Println(text)
			return nil
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}
