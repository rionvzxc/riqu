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

	var bodyErr error
	err = riqu.New().GET(*targetURL).
		Transport(transport).
		OnSuccess(func(response *riqu.QResponse) {
			var text string
			text, bodyErr = response.Text()
			if bodyErr == nil {
				fmt.Println("Status:", response.StatusCode)
				fmt.Println(text)
			}
		}).Send()
	if err != nil {
		log.Fatal(err)
	}
	if bodyErr != nil {
		log.Fatal(bodyErr)
	}
}
