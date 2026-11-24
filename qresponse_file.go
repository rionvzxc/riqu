package riqu

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

// File saves the response body using the supplied file options.
func (this *QResponse) File(options *TFileOptions) error {
	if this == nil {
		return fmt.Errorf("riqu: response is nil")
	}
	if this.bodyRead {
		return fmt.Errorf("riqu: response body already read")
	}
	if this.body == nil {
		return fmt.Errorf("riqu: response body unavailable")
	}
	if options == nil {
		options = &TFileOptions{}
	}
	// FILENAME
	filename := options.Filename

	if filename == "" {
		_, params, err := mime.ParseMediaType(
			this.resHeader.Get("Content-Disposition"),
		)
		if err == nil {
			filename = params["filename"]
		}
	}
	if filename == "" {
		filename = options.DefaultFilename
	}
	if filename == "" {
		parsedURL, err := url.Parse(this.resURL)

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
		contentType := this.resHeader.Get("Content-Type")
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
			err = errors.Join(err, fmt.Errorf("riqu: remove temporary download: %w", removeErr))
		}
		return err
	}
	return nil
}
