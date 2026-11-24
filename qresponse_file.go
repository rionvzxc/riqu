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
	// VALIDATION - Check that the response body is available and not consumed.
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
	// FILENAME - Prefer explicit options, then response headers and URL fallbacks.
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
	// EXTENSION - Infer an extension from Content-Type when the filename has none.
	if filepath.Ext(filename) == "" {
		contentType := this.resHeader.Get("Content-Type")
		mediaType, _, _ := mime.ParseMediaType(contentType)
		extensions, _ := mime.ExtensionsByType(mediaType)

		if len(extensions) > 0 {
			filename += extensions[0]
		}
	}
	// DIRECTORY - Create the requested directory and resolve the destination.
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
	isFileSaved := false
	defer func() {
		file.Close()

		if !isFileSaved {
			os.Remove(file.Name())
		}
	}()
	// COPY - Consume the body into the temporary file and collect close errors.
	err = this.consumeBody(func(reader io.Reader) error {
		_, copyErr := io.Copy(file, reader)
		return errors.Join(copyErr, file.Close())
	})

	// REPLACE - Publish the completed file only after copying and closing succeed.
	if err == nil {
		err = os.Rename(file.Name(), filename)
	}
	// CLEANUP - Remove the temporary file if copying or replacement failed.
	if err != nil {
		removeErr := os.Remove(file.Name())

		if removeErr != nil && !os.IsNotExist(removeErr) {
			err = errors.Join(err, fmt.Errorf("riqu: remove temporary download: %w", removeErr))
		}
		return err
	}
	isFileSaved = true
	return nil
}
