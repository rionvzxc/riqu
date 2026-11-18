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
	"path"
	"path/filepath"
)

// QResponse contains the metadata and body of an HTTP response.
type QResponse struct {
	// StatusCode is the numeric HTTP status code.
	StatusCode int
	// Status is the HTTP status text.
	Status string
	// Header contains the response headers.
	Header http.Header
	// Cookies contains cookies set by the response.
	Cookies []*http.Cookie
	// RequestUrl is the URL used for this response.
	RequestUrl string
	// RequestMethod is the HTTP method used for this response.
	RequestMethod string
	bodyRead      bool
	body          io.ReadCloser
}
// Data reads and closes the response body.
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
// Text reads the response body as a string.
func (this *QResponse) Text() (string, error) {
	rawData, err := this.Data()
	if err != nil {
		return "", err
	}
	return string(rawData), nil
}
// JSON decodes the response body into target.
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
// File saves the response body using the supplied file options.
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
			filename = path.Base(parsedURL.Path)

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
	// Stage the download beside its destination so it can be renamed in place.
	file, err := os.CreateTemp(filepath.Dir(filename), ".riqu-download-*")

	if err != nil {
		return err
	}
	this.bodyRead = true
	_, copyErr := io.Copy(file, this.body)
	fileCloseErr := file.Close()
	bodyCloseErr := this.body.Close()
	err = errors.Join(copyErr, fileCloseErr, bodyCloseErr)

	if err == nil {
		err = os.Rename(file.Name(), filename)
	}
	if err != nil {
		removeErr := os.Remove(file.Name())

		if removeErr != nil && !os.IsNotExist(removeErr) {
			err = errors.Join(err, fmt.Errorf("remove temporary download: %w", removeErr))
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
