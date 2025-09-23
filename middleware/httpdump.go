package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/millken/inertia"
)

const maxDumpBody = 1 << 20 // 1MB

// HTTPDump is a middleware that dumps the request and response to the console.
func HTTPDump() inertia.HandlerFunc {
	return func(c *inertia.Context) {
		r := c.Request
		parts := []string{"curl", "-i", "-X", r.Method}

		for k, v := range r.Header {
			ck := http.CanonicalHeaderKey(k)
			if ck != "Host" && ck != "User-Agent" {
				for _, vv := range v {
					parts = append(parts, "-H", fmt.Sprintf("'%s: %s'", k, vv))
				}

			}
		}

		if r.Method == "POST" || r.Method == "PATCH" || r.Method == "PUT" {
			var (
				b   []byte
				err error
			)

			if r.Body != nil {
				limited := io.LimitReader(r.Body, maxDumpBody+1)
				b, err = io.ReadAll(limited)

				// close original body if possible
				if closer, ok := r.Body.(io.Closer); ok {
					closer.Close()
				}

				// Always restore body so downstream handlers can read it.
				r.Body = io.NopCloser(bytes.NewBuffer(b))
				r.ContentLength = int64(len(b))
			} else {
				b = nil
			}

			if err != nil {
				fmt.Println("dumphttp read body error:", err)
			} else if len(b) == 0 {
				// nothing to append but ensure body restored (already done)
			} else if int64(len(b)) > maxDumpBody {
				parts = append(parts, "-d '[truncated]'")
			} else {
				parts = append(parts, "-d '"+string(b)+"'")
			}
		}

		scheme := "http"
		if r.TLS != nil {
			scheme += "s"
		}

		parts = append(parts, scheme+"://"+r.Host+r.URL.String())
		fmt.Println(strings.Join(parts, " "))

		c.Next()
	}
}
