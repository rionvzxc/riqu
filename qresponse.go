package riqu

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type QResponse struct {
	StatusCode    int
	Status        string
	Header        http.Header
	Cookies       []*http.Cookie
	RequestUrl    string
	RequestMethod string
	bodyRead      bool
	body io.ReadCloser
}
func (this *QResponse) Data() ([]byte, error) {
	if this == nil {
		return nil, fmt.Errorf("Response is nil")
	}
	if this.bodyRead {
		return nil, fmt.Errorf("Response body already read")
	}
	if this.body == nil {
		return nil, fmt.Errorf("Response body unavailable")
	}
	this.bodyRead = true
	data, readErr := io.ReadAll(this.body)
	closeErr := this.body.Close()
	return data, errors.Join(readErr, closeErr)
}
func (this *QResponse) Text() (string, error) {
	rawData, err := this.Data()
	if err != nil {
		return "", err
	}
	return string(rawData), nil
}
func (this *QResponse) JSON(target any) error {
	if this == nil {
		return fmt.Errorf("Response is nil")
	}
	if this.bodyRead {
		return fmt.Errorf("Response body already read")
	}
	if this.body == nil {
		return fmt.Errorf("Response body unavailable")
	}
	this.bodyRead = true
	decodeErr := json.NewDecoder(this.body).Decode(target)
	closeErr := this.body.Close()
	return errors.Join(decodeErr, closeErr)
}
func (this *QResponse) File(options *TFileOptions) error {
	if this == nil {
		return fmt.Errorf("Response is nil")
	}
	if this.bodyRead {
		return fmt.Errorf("Response body already read")
	}
	if this.body == nil {
		return fmt.Errorf("Response body unavailable")
	}
	if options == nil {
		options = &TFileOptions{}
	}
	// FILENAME
	filename := options.Filename

	if filename == "" {
		_, params, err := mime.ParseMediaType(
			this.Header.Get("Content-Disposition"),
		)
		if err == nil {
			filename = params["filename"]
		}
	}
	if filename == "" {
		filename = options.DefaultFilename
	}
	if filename == "" {
		parsedURL, err := url.Parse(this.RequestUrl)

		if err == nil {
			filename = filepath.Base(parsedURL.Path)

			if filename == "." || filename == "/" {
				filename = ""
			}
		}
	}
	if filename == "" {
		filename = "download"
	}
	filename = filepath.Base(filename)
	// EXTENSION
	if filepath.Ext(filename) == "" {
		contentType := this.Header.Get("Content-Type")
		mediaType, _, _ := mime.ParseMediaType(contentType)
		extensions, _ := mime.ExtensionsByType(mediaType)

		if len(extensions) > 0 {
			filename += extensions[0]
		}
	}
	// DIRECTORY
	if options.Dir != "" {
		if err := os.MkdirAll(options.Dir, 0755); err != nil {
			return err
		}
		filename = filepath.Join(options.Dir, filename)
	}
	// CREATE FILE
	file, err := os.Create(filename)

	if err != nil {
		return err
	}
	this.bodyRead = true
	_, copyErr := io.Copy(file, this.body)
	fileCloseErr := file.Close()
	bodyCloseErr := this.body.Close()
	err = errors.Join(copyErr, fileCloseErr, bodyCloseErr)

	if err != nil {
		removeErr := os.Remove(filename)
		
		if removeErr != nil && !os.IsNotExist(removeErr) {
			err = errors.Join(err, fmt.Errorf("remove incomplete file: %w", removeErr))
		}
		return err
	}
	return nil
}
func (this *QResponse) close() error {
	if this == nil || this.body == nil {
		return nil
	}
	this.bodyRead = true
	return this.body.Close()
}
