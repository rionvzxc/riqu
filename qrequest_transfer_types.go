package riqu

import (
	"io"
	"sync"
	"time"
)

type tTransferOptions struct {
	start    func(TTransferInfo)
	progress func(TTransferInfo)
	end      func(TTransferInfo)
	interval time.Duration
}

func (this tTransferOptions) enabled() bool {
	return this.start != nil || this.progress != nil || this.end != nil
}

type tTransferBody struct {
	body         io.ReadCloser
	options      tTransferOptions
	info         TTransferInfo
	mutex        sync.Mutex
	isStarted      bool
	isEnded        bool
	isEOF          bool
	readErr      error
	lastProgress time.Time
	closeOnce    sync.Once
	closeErr     error
	prevBody     *tTransferBody
	startOnce    sync.Once
}

func (this *tTransferBody) Read(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	this.startOnce.Do(func() {
		// A prepared replay is a new transfer only when the transport reads it.
		for prevBody := this.prevBody; prevBody != nil; prevBody = prevBody.prevBody {
			closeErr := prevBody.Close()
			prevBody.finish(closeErr, true)
		}
	})
	this.start()
	bytesRead, err := this.body.Read(data)
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if !this.isEnded {
		this.info.Transferred += int64(bytesRead)

		if err == io.EOF {
			this.isEOF = true
		} else if err != nil {
			this.readErr = err
		}
		if bytesRead > 0 && this.options.progress != nil && time.Since(this.lastProgress) >= this.options.interval {
			this.lastProgress = time.Now()
			this.options.progress(this.info)
		}
	}
	return bytesRead, err
}

func (this *tTransferBody) start() {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if !this.isStarted && !this.isEnded {
		this.isStarted = true
		this.lastProgress = time.Now()
		if this.options.start != nil {
			this.options.start(this.info)
		}
	}
}

func (this *tTransferBody) Close() error {
	this.closeOnce.Do(func() { this.closeErr = this.body.Close() })
	return this.closeErr
}

func (this *tTransferBody) finish(err error, isUpload bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.isEnded {
		return
	}
	this.isEnded = true

	if !this.isStarted {
		return
	}
	if err == nil {
		err = this.readErr
	}
	if isUpload && err == nil && !this.isEOF && (this.info.Total < 0 || this.info.Transferred < this.info.Total) {
		err = io.ErrUnexpectedEOF
	}
	this.info.Err = err

	if this.options.end != nil {
		this.options.end(this.info)
	}
}
