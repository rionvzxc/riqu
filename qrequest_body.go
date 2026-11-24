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

// BodyReader sets a one-shot body reader owned by the caller.
func (this *QRequest) BodyReader(reader io.Reader) *QRequest {
	this.body = tRequestBody{bodyType: vRequestBodyReader, value: &tRequestReader{reader: reader}}
	return this
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
	// ENCODING - Convert the current body value into bytes or a multipart description.
	prepared := tPreparedBody{contentType: this.body.contentType}

	switch this.body.bodyType {
	case vRequestBodyReader:
		reader, isValidType := this.body.value.(*tRequestReader)

		if !isValidType || reader == nil || reader.reader == nil {
			return tPreparedBody{}, errors.New("riqu: body reader is nil")
		}
		prepared.reader = reader
	case vRequestBodyData:
		data, isValidType := this.body.value.([]byte)

		if !isValidType && this.body.value != nil {
			return tPreparedBody{}, errors.New("riqu: invalid raw body")
		}
		prepared.data = data
	case vRequestBodyText, vRequestBodyHTML:
		value, isValidType := this.body.value.(string)

		if !isValidType {
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
		formMap, isValidType := this.body.value.(TValues)

		if !isValidType {
			return tPreparedBody{}, errors.New("riqu: invalid form body")
		}
		form := url.Values{}

		for key, value := range formMap {
			form[key] = append([]string(nil), value...)
		}
		prepared.data = []byte(form.Encode())
	case vRequestBodyMultipart:
		formData, isValidType := this.body.value.(TMultipart)

		if !isValidType {
			return tPreparedBody{}, errors.New("riqu: invalid multipart body")
		}
		prepared.multipart = &formData
	default:
		return tPreparedBody{}, errors.New("riqu: invalid request body type")
	}
	return prepared, nil
}
func (this *QRequest) multipartBodyReader(formData TMultipart) (io.ReadCloser, string, error) {
	// CONTENT TYPE - Validate an explicit multipart header and read its parameters.
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
	// BOUNDARY - Keep the header and streamed multipart body on the same boundary.
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
	// STREAM - Produce multipart data as the HTTP transport reads from the pipe.
	go func() {
		var writeErr error
		// FIELDS - Write all values for each form field.
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
		// FILES - Open, copy, and close each file without buffering its full contents.
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
		// FINISH - Write the closing boundary or propagate the producer error.
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
