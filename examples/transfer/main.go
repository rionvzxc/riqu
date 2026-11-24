package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rionvzxc/riqu"
)

func main() {
	riqu.New().
		POST("https://httpbin.org/post").
		Text(strings.Repeat("riqu ", 4096)).
		OnUploadStart(func(info riqu.TTransferInfo) {
			fmt.Println("Upload started:", info.URL)
		}).
		OnUploadProgress(time.Second, func(info riqu.TTransferInfo) {
			fmt.Println("Uploaded:", info.Transferred, "/", info.Total)
		}).
		OnUploadEnd(func(info riqu.TTransferInfo) {
			fmt.Println("Upload ended:", info.Transferred, info.Err)
		}).
		OnDownloadStart(func(info riqu.TTransferInfo) {
			fmt.Println("Download started:", info.URL)
		}).
		OnDownloadProgress(time.Second, func(info riqu.TTransferInfo) {
			fmt.Println("Downloaded:", info.Transferred, "/", info.Total)
		}).
		OnDownloadEnd(func(info riqu.TTransferInfo) {
			fmt.Println("Download ended:", info.Transferred, info.Err)
		}).
		OnResponse(func(res *riqu.QResponse) error {
			_, err := res.WriteTo(io.Discard)
			return err
		}).
		OnError(func(err error) {
			fmt.Println("Error:", err)
		}).
		Send()
}
