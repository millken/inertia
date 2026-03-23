package web

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/millken/inertia/router"
)

type Header struct {
	Key   string
	Value string
}

type Query string

func (q Query) Param(name string) string {
	query := strings.ReplaceAll(string(q), "+", " ")
	for pair := range strings.SplitSeq(query, "&") {
		if pair == "" {
			continue
		}
		key := pair
		value := ""
		if equal := strings.IndexByte(pair, '='); equal != -1 {
			key = pair[:equal]
			value = pair[equal+1:]
		}
		if decodeComponent(key) == name {
			return decodeComponent(value)
		}
	}
	return ""
}

type Request struct {
	reader     bufio.Reader
	bodyReader byteSliceReader
	readBufferSize int
	ctx        context.Context
	scheme     string
	host       string
	method     string
	path       string
	query      Query
	remote     string
	headers    []Header
	params     Params
	body       []byte
	formCache  url.Values
	targetBuf  []byte
	headerBuf  []byte
	length     int
	consumed   int
}

func (req *Request) setReadBufferSize(size int) {
	if size <= 0 {
		size = 4096
	}
	if req.readBufferSize == size {
		return
	}
	req.reader = *bufio.NewReaderSize(nil, size)
	req.readBufferSize = size
}

func (req *Request) reset() {
	req.ctx = context.Background()
	req.scheme = ""
	req.host = ""
	req.method = ""
	req.path = ""
	req.query = ""
	req.remote = ""
	req.headers = req.headers[:0]
	req.params = req.params[:0]
	req.body = req.body[:0]
	req.formCache = nil
	req.targetBuf = req.targetBuf[:0]
	req.headerBuf = req.headerBuf[:0]
	req.length = -1
	req.consumed = 0
	req.bodyReader.Reset(nil)
	req.reader.Reset(nil)
}

func (req *Request) setContext(ctx context.Context) {
	if ctx == nil {
		req.ctx = context.Background()
		return
	}
	req.ctx = ctx
}

func (req *Request) setRemoteAddr(remote string) {
	req.remote = remote
}

func (req *Request) setURLParts(scheme string, host string, path string, query Query) {
	req.scheme = scheme
	req.host = host
	req.path = path
	req.query = query
	if req.path == "" {
		req.path = "/"
	}
	if req.host == "" {
		req.host = req.Header("Host")
	}
}

func (req *Request) setURLPartsBytes(scheme []byte, host []byte, path []byte, query []byte) {
	req.targetBuf = req.targetBuf[:0]
	req.targetBuf, req.scheme = appendOwnedString(req.targetBuf, scheme)
	req.targetBuf, req.host = appendOwnedString(req.targetBuf, host)
	req.targetBuf, req.path = appendOwnedString(req.targetBuf, path)
	if req.path == "" {
		req.path = "/"
	}
	var queryString string
	req.targetBuf, queryString = appendOwnedString(req.targetBuf, query)
	req.query = Query(queryString)
	if req.host == "" {
		req.host = req.Header("Host")
	}
}

func (req *Request) addParameter(key string, value string) {
	req.params = append(req.params, router.Parameter{Key: key, Value: value})
}

func (req *Request) appendHeaderBytes(key []byte, value []byte) {
	keyString, common := commonHeaderKey(key)
	if !common {
		req.headerBuf, keyString = appendOwnedString(req.headerBuf, key)
	}
	var valueString string
	req.headerBuf, valueString = appendOwnedString(req.headerBuf, value)
	req.headers = append(req.headers, Header{Key: keyString, Value: valueString})
}

func (req *Request) ensureContentLength() int {
	if req.length >= 0 {
		return req.length
	}
	if len(req.body) > 0 {
		req.length = len(req.body)
		return req.length
	}
	contentLength := req.Header("Content-Length")
	if contentLength == "" {
		req.length = 0
		return req.length
	}
	req.length, _ = strconv.Atoi(contentLength)
	if req.length < 0 {
		req.length = 0
	}
	return req.length
}

func (req *Request) ContentLength() int { return req.ensureContentLength() }

func (req *Request) Context() context.Context {
	if req.ctx == nil {
		return context.Background()
	}
	return req.ctx
}

func (req *Request) Header(key string) string {
	for _, header := range req.headers {
		if strings.EqualFold(header.Key, key) {
			return header.Value
		}
	}
	return ""
}

func (req *Request) Headers() []Header { return slices.Clone(req.headers) }

func (req *Request) Host() string {
	if req.host != "" {
		return req.host
	}
	return req.Header("Host")
}

func (req *Request) Method() string { return req.method }
func (req *Request) Param(name string) string { return req.params.Get(name) }
func (req *Request) Params() Params { return append(Params(nil), req.params...) }
func (req *Request) Query() Query { return req.query }
func (req *Request) RemoteAddr() string { return req.remote }
func (req *Request) Path() string { return req.path }
func (req *Request) RawQuery() string { return string(req.query) }

func (req *Request) Read(p []byte) (n int, err error) {
	if len(req.body) > 0 && req.consumed == 0 {
		req.bodyReader.Reset(req.body)
		req.reader.Reset(&req.bodyReader)
	}
	if req.ensureContentLength() == 0 {
		return 0, io.EOF
	}
	n, err = req.reader.Read(p)
	req.consumed += n
	if req.consumed < req.length {
		return n, err
	}
	return n - (req.consumed - req.length), io.EOF
}

func (req *Request) Scheme() string { return req.scheme }
func (req *Request) UserAgent() string { return req.Header("User-Agent") }
func (req *Request) Body() []byte { return bytes.Clone(req.body) }
func (req *Request) PostForm(key string) string { value, _ := req.GetPostForm(key); return value }

func (req *Request) DefaultPostForm(key, defaultValue string) string {
	if value, ok := req.GetPostForm(key); ok {
		return value
	}
	return defaultValue
}

func (req *Request) GetPostForm(key string) (string, bool) {
	if req.formCache != nil {
		if values, ok := req.formCache[key]; ok && len(values) > 0 {
			return values[0], true
		}
		return "", false
	}
	if len(req.body) == 0 {
		return "", false
	}
	if value, ok := findPostFormValueBytes(req.body, key); ok {
		return value, true
	}
	return "", false
}

func (req *Request) PostFormArray(key string) []string { values, _ := req.GetPostFormArray(key); return values }

func (req *Request) GetPostFormArray(key string) ([]string, bool) {
	req.initFormCache()
	values, ok := req.formCache[key]
	return values, ok
}

func (req *Request) initFormCache() {
	if req.formCache != nil {
		return
	}
	req.formCache = make(url.Values)
	if len(req.body) == 0 {
		return
	}
	parseValuesBytesInto(req.formCache, req.body)
}

func (req *Request) setOwnedBody(body []byte) {
	req.body = body
	req.bindBodyReader()
}

func (req *Request) prepareBody(length int) []byte {
	if cap(req.body) < length {
		req.body = make([]byte, length)
	} else {
		req.body = req.body[:length]
	}
	req.length = length
	req.consumed = 0
	req.formCache = nil
	return req.body
}

func (req *Request) bindBodyReader() {
	req.length = len(req.body)
	req.consumed = 0
	req.formCache = nil
	req.bodyReader.Reset(req.body)
	req.reader.Reset(&req.bodyReader)
}

func ParseURL(target string) (scheme string, host string, path string, query Query) {
	schemePos := strings.Index(target, "://")
	if schemePos != -1 {
		scheme = target[:schemePos]
		target = target[schemePos+3:]
	}
	pathPos := strings.IndexByte(target, '/')
	if pathPos != -1 {
		host = target[:pathPos]
		target = target[pathPos:]
	}
	queryPos := strings.IndexByte(target, '?')
	if queryPos != -1 {
		path = target[:queryPos]
		query = Query(target[queryPos+1:])
		return
	}
	path = target
	return
}

func decodeComponent(value string) string {
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}

func parseValuesBytesInto(dst url.Values, raw []byte) {
	start := 0
	for start <= len(raw) {
		end := len(raw)
		if offset := bytes.IndexByte(raw[start:], '&'); offset >= 0 {
			end = start + offset
		}
		pair := raw[start:end]
		if len(pair) > 0 {
			key := pair
			value := []byte(nil)
			if equal := bytes.IndexByte(pair, '='); equal != -1 {
				key = pair[:equal]
				value = pair[equal+1:]
			}
			decodedKey := decodeComponentBytes(key)
			dst[decodedKey] = append(dst[decodedKey], decodeComponentBytes(value))
		}
		if end == len(raw) {
			return
		}
		start = end + 1
	}
}

func decodeComponentBytes(value []byte) string {
	if len(value) == 0 {
		return ""
	}
	if bytes.IndexByte(value, '%') == -1 && bytes.IndexByte(value, '+') == -1 {
		return b2s(value)
	}
	return decodeComponent(b2s(value))
}

func findPostFormValueBytes(raw []byte, key string) (string, bool) {
	start := 0
	for start <= len(raw) {
		end := len(raw)
		if offset := bytes.IndexByte(raw[start:], '&'); offset >= 0 {
			end = start + offset
		}
		pair := raw[start:end]
		if len(pair) > 0 {
			name := pair
			value := []byte(nil)
			if equal := bytes.IndexByte(pair, '='); equal != -1 {
				name = pair[:equal]
				value = pair[equal+1:]
			}
			if postFormKeyMatches(name, key) {
				return decodeComponentBytes(value), true
			}
		}
		if end == len(raw) {
			return "", false
		}
		start = end + 1
	}
	return "", false
}

func postFormKeyMatches(raw []byte, key string) bool {
	if bytes.IndexByte(raw, '%') == -1 && bytes.IndexByte(raw, '+') == -1 {
		return b2s(raw) == key
	}
	return decodeComponentBytes(raw) == key
}

func appendOwnedString(buffer []byte, value []byte) ([]byte, string) {
	if len(value) == 0 {
		return buffer, ""
	}
	start := len(buffer)
	buffer = append(buffer, value...)
	segment := buffer[start : start+len(value)]
	return buffer, b2s(segment)
}

func commonHeaderKey(key []byte) (string, bool) {
	switch {
	case bytes.EqualFold(key, s2b("Host")):
		return "Host", true
	case bytes.EqualFold(key, s2b("Connection")):
		return "Connection", true
	case bytes.EqualFold(key, s2b("Content-Length")):
		return "Content-Length", true
	case bytes.EqualFold(key, s2b("Content-Type")):
		return "Content-Type", true
	case bytes.EqualFold(key, s2b("User-Agent")):
		return "User-Agent", true
	case bytes.EqualFold(key, s2b("X-Forwarded-For")):
		return "X-Forwarded-For", true
	case bytes.EqualFold(key, s2b("X-Real-IP")):
		return "X-Real-IP", true
	default:
		return "", false
	}
}