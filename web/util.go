package web

import (
	"bytes"
	"encoding/json"
	"sync"
	"unsafe"
)

func b2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func s2b(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

var jsonBufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

const maxPooledJSONBufferCapacity = 256 * 1024

func jsonMarshal(v any, escapeHTML bool) ([]byte, error) {
	buf := jsonBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		buf.Reset()
		if buf.Cap() <= maxPooledJSONBufferCapacity {
			jsonBufferPool.Put(buf)
		}
	}()
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(escapeHTML)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] == '\n' {
		buf.Truncate(buf.Len() - 1)
	}
	return bytes.Clone(buf.Bytes()), nil
}