package riqu

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
)

type tRequestBodyType uint

const (
	vRequestBodyData tRequestBodyType = iota
	vRequestBodyText
	vRequestBodyHTML
	vRequestBodyJSON
	vRequestBodyForm
	vRequestBodyMultipart
)

type tPreparedBody struct {
	data        []byte
	contentType string
	multipart   *TMultipart
}
// Data sets a raw request body.
func (this *QRequest) Data(data []byte) *QRequest {
	this.body = tRequestBody{
		bodyType: vRequestBodyData,
		value:    data,
	}
	return this
}
// Text sets a plain text request body.
func (this *QRequest) Text(text string) *QRequest {
	this.body = tRequestBody{
		bodyType:    vRequestBodyText,
		value:       text,
		contentType: "text/plain",
	}
	return this
}
// HTML sets an HTML request body.
func (this *QRequest) HTML(html string) *QRequest {
	this.body = tRequestBody{
		bodyType:    vRequestBodyHTML,
		value:       html,
		contentType: "text/html",
	}
	return this
}
// JSON sets a request body that is JSON-encoded when sent.
func (this *QRequest) JSON(jsonData any) *QRequest {
	this.body = tRequestBody{
		bodyType:    vRequestBodyJSON,
		value:       jsonData,
		contentType: "application/json",
	}
	return this
}
// Form sets a URL-encoded form request body.
func (this *QRequest) Form(formMap TValues) *QRequest {
	this.body = tRequestBody{
		bodyType:    vRequestBodyForm,
		value:       formMap.Clone(),
		contentType: "application/x-www-form-urlencoded",
	}
	return this
}
// FormMultipart sets a multipart form body with fields and file paths.
func (this *QRequest) FormMultipart(multipart TMultipart) *QRequest {
	this.body = tRequestBody{
		bodyType: vRequestBodyMultipart,
		value: TMultipart{
			Form: multipart.Form.Clone(),
			File: multipart.File.Clone(),
		},
	}
	return this
}
func (this *QRequest) prepareBody() (tPreparedBody, error) {
	prepared := tPreparedBody{contentType: this.body.contentType}

	switch this.body.bodyType {
	case vRequestBodyData:
		data, ok := this.body.value.([]byte)

		if !ok && this.body.value != nil {
			return tPreparedBody{}, errors.New("riqu: invalid raw body")
		}
		prepared.data = data
	case vRequestBodyText, vRequestBodyHTML:
		value, ok := this.body.value.(string)

		if !ok {
			return tPreparedBody{}, errors.New("riqu: invalid request body")
		}
		prepared.data = []byte(value)
	case vRequestBodyJSON:
		data, err := json.Marshal(this.body.value)

		if err != nil {
			return tPreparedBody{}, err
		}
		prepared.data = data
	case vRequestBodyForm:
		formMap, ok := this.body.value.(TValues)

		if !ok {
			return tPreparedBody{}, errors.New("riqu: invalid form body")
		}
		form := url.Values{}

		for key, value := range formMap {
			form[key] = append([]string(nil), value...)
		}
		prepared.data = []byte(form.Encode())
	case vRequestBodyMultipart:
		formData, ok := this.body.value.(TMultipart)

		if !ok {
			return tPreparedBody{}, errors.New("riqu: invalid multipart body")
		}
		prepared.multipart = &formData
	default:
		return tPreparedBody{}, errors.New("riqu: invalid request body type")
	}
	return prepared, nil
}
func (this *QRequest) multipartBodyReader(formData TMultipart) (io.ReadCloser, string, error) {
	contentType := this.header.Get("Content-Type")
	var params map[string]string

	if contentType != "" {
		mediaType, parsedParams, err := mime.ParseMediaType(contentType)

		if err != nil {
			return nil, "", fmt.Errorf("riqu: invalid multipart Content-Type: %w", err)
		}
		if mediaType != "multipart/form-data" {
			return nil, "", fmt.Errorf("riqu: invalid multipart Content-Type %q: expected multipart/form-data", mediaType)
		}
		params = parsedParams
	}
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)

	if contentType == "" {
		contentType = writer.FormDataContentType()
	} else if params["boundary"] == "" {
		params["boundary"] = writer.Boundary()
		contentType = mime.FormatMediaType("multipart/form-data", params)
	} else if err := writer.SetBoundary(params["boundary"]); err != nil {
		pipeReader.Close()
		pipeWriter.CloseWithError(err)
		return nil, "", err
	}

	go func() {
		var writeErr error

		for key, values := range formData.Form {
			for _, value := range values {
				if err := writer.WriteField(key, value); err != nil {
					writeErr = err
					break
				}
			}
			if writeErr != nil {
				break
			}
		}
		if writeErr == nil {
			for key, paths := range formData.File {
				for _, path := range paths {
					file, err := os.Open(path)

					if err != nil {
						writeErr = fmt.Errorf("riqu: open multipart file %q: %w", path, err)
						break
					}
					part, err := writer.CreateFormFile(key, filepath.Base(path))

					if err != nil {
						writeErr = errors.Join(fmt.Errorf("riqu: create multipart file %q: %w", path, err), file.Close())
						break
					}
					_, copyErr := io.Copy(part, file)
					closeErr := file.Close()

					if copyErr != nil {
						copyErr = fmt.Errorf("riqu: copy multipart file %q: %w", path, copyErr)
					}
					if closeErr != nil {
						closeErr = fmt.Errorf("riqu: close multipart file %q: %w", path, closeErr)
					}
					if err := errors.Join(copyErr, closeErr); err != nil {
						writeErr = err
						break
					}
				}
				if writeErr != nil {
					break
				}
			}
		}
		if writeErr == nil {
			writeErr = writer.Close()
		}
		if writeErr != nil {
			pipeWriter.CloseWithError(writeErr)
			return
		}
		pipeWriter.Close()
	}()
	return pipeReader, contentType, nil
}
