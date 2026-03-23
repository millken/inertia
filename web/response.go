package web

import (
	"bytes"
	"io"
	"slices"
	"strconv"
	"strings"
	"sync"
)

const noWritten = -1
const maxPooledHTTPBufferCap = 64 * 1024

var httpBufferPool = sync.Pool{
	New: func() any {
		buffer := make([]byte, 0, 1024)
		return &buffer
	},
}

type Response struct {
	body    []byte
	headers []Header
	status  uint16
	size    int
	written bool
	writeBufferSize int
}

func (res *Response) reset() {
	res.body = res.body[:0]
	res.headers = res.headers[:0]
	res.status = 200
	res.size = noWritten
	res.written = false
}

func (res *Response) stealFrom(src *Response) {
	oldBody := res.body[:0]
	oldHeaders := res.headers[:0]
	res.body = src.body
	res.headers = src.headers
	res.status = src.status
	res.size = src.size
	res.written = src.written
	src.body = oldBody
	src.headers = oldHeaders
	src.status = 200
	src.size = noWritten
	src.written = false
}

func (res *Response) Body() []byte { return res.body }

func (res *Response) DeleteHeader(key string) {
	for index, header := range res.headers {
		if strings.EqualFold(header.Key, key) {
			res.headers = append(res.headers[:index], res.headers[index+1:]...)
			return
		}
	}
}

func (res *Response) Header(key string) string {
	for _, header := range res.headers {
		if strings.EqualFold(header.Key, key) {
			return header.Value
		}
	}
	return ""
}

func (res *Response) Headers() []Header { return slices.Clone(res.headers) }

func (res *Response) SetHeader(key string, value string) {
	for index, header := range res.headers {
		if strings.EqualFold(header.Key, key) {
			res.headers[index].Value = value
			return
		}
	}
	res.headers = append(res.headers, Header{Key: key, Value: value})
}

func (res *Response) SetBody(body []byte) {
	if body == nil {
		res.body = nil
	} else {
		res.body = bytes.Clone(body)
	}
	if len(res.body) > 0 || res.written {
		res.written = true
		res.size = len(res.body)
		return
	}
	res.size = noWritten
}

func (res *Response) SetStatus(status int) { res.status = uint16(status) }
func (res *Response) Size() int { return res.size }
func (res *Response) Status() int { return int(res.status) }

func (res *Response) WriteHeader(status int) {
	if res.written {
		return
	}
	res.status = uint16(status)
	res.written = true
	res.size = 0
}

func (res *Response) Written() bool { return res.written }

func (res *Response) Write(body []byte) (int, error) {
	if !res.written {
		res.written = true
		res.size = 0
	}
	res.body = append(res.body, body...)
	res.size += len(body)
	return len(body), nil
}

func (res *Response) WriteString(body string) (int, error) {
	if !res.written {
		res.written = true
		res.size = 0
	}
	res.body = append(res.body, body...)
	res.size += len(body)
	return len(body), nil
}

func (res *Response) AppendHTTP(dst []byte) []byte {
	status := res.Status()
	if status == 0 {
		status = 200
	}
	dst = append(dst, "HTTP/1.1 "...)
	dst = strconv.AppendInt(dst, int64(status), 10)
	dst = append(dst, "\r\nContent-Length: "...)
	dst = strconv.AppendInt(dst, int64(len(res.body)), 10)
	dst = append(dst, "\r\n"...)
	for _, header := range res.headers {
		dst = append(dst, header.Key...)
		dst = append(dst, ':', ' ')
		dst = append(dst, header.Value...)
		dst = append(dst, '\r', '\n')
	}
	dst = append(dst, '\r', '\n')
	dst = append(dst, res.body...)
	return dst
}

func (res *Response) WriteTo(writer io.Writer) (int64, error) {
	bufferRef := httpBufferPool.Get().(*[]byte)
	payloadBuffer := (*bufferRef)[:0]
	if res.writeBufferSize > 0 && cap(payloadBuffer) < res.writeBufferSize {
		payloadBuffer = make([]byte, 0, res.writeBufferSize)
	}
	payload := res.AppendHTTP(payloadBuffer)
	n, err := writer.Write(payload)
	if cap(payload) <= maxPooledHTTPBufferCap {
		*bufferRef = payload[:0]
		httpBufferPool.Put(bufferRef)
	}
	return int64(n), err
}