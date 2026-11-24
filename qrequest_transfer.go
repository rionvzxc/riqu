package riqu

import "time"

// OnUploadStart runs when the transport starts reading a request body.
// Upload callbacks run on the transport's reading goroutine.
func (this *QRequest) OnUploadStart(callback func(TTransferInfo)) *QRequest {
	this.uploadOptions.start = callback
	return this
}

// OnUploadProgress reports body bytes read by the transport, at most once per interval.
// Zero reports every read; a negative interval is invalid. Callbacks run synchronously.
func (this *QRequest) OnUploadProgress(interval time.Duration, callback func(TTransferInfo)) *QRequest {
	this.uploadOptions.interval = interval
	this.uploadOptions.progress = callback
	return this
}

// OnUploadEnd reports the end of an upload attempt, not server acceptance.
func (this *QRequest) OnUploadEnd(callback func(TTransferInfo)) *QRequest {
	this.uploadOptions.end = callback
	return this
}

// OnDownloadStart runs when a response body starts being read by a consumer.
func (this *QRequest) OnDownloadStart(callback func(TTransferInfo)) *QRequest {
	this.downloadOptions.start = callback
	return this
}

// OnDownloadProgress reports consumed response bytes, at most once per interval.
// Zero reports every read; a negative interval is invalid. Callbacks run synchronously.
func (this *QRequest) OnDownloadProgress(interval time.Duration, callback func(TTransferInfo)) *QRequest {
	this.downloadOptions.interval = interval
	this.downloadOptions.progress = callback
	return this
}

// OnDownloadEnd reports body processing and close errors, before any File rename.
// It is only called for a transfer that has started; it does not indicate request success.
func (this *QRequest) OnDownloadEnd(callback func(TTransferInfo)) *QRequest {
	this.downloadOptions.end = callback
	return this
}
