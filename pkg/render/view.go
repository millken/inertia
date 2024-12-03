package render

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"unsafe"
)

type Ctx struct {
	// value map[string]any
}

func FromContext(ctx context.Context) *View {
	if r, ok := ctx.Value(Ctx{}).(*View); ok {
		return r
	}
	panic("render not found in context")
}

type View struct {
	render  *Render
	request *http.Request
	writer  http.ResponseWriter
	data    map[string]any
}

func (v *View) Set(key string, value any) {
	v.data[key] = value
}
func (v *View) Render(view string) error {
	if _, ok := v.render.compiled.Load(view); ok {
		return v.execute(view)
	}
	// Check if the view has been compiled
	if _, err := v.render.svelteFS.Open(view + ".svelte"); err != nil {
		return fmt.Errorf("view %q not found", view)
	}
	if _, err := v.render.compileFS.Open(view + ".js"); err != nil {
		return fmt.Errorf("view %q not compiled", view)
	}
	v.render.compiled.Store(view, struct{}{})
	return v.execute(view)
}

func (v *View) execute(view string) error {
	if v.render.rootTemplateHTML == nil {
		return fmt.Errorf("root template not set")
	}
	v.data["View"] = view

	if v.request.Header.Get("X-Pjax") == "true" {
		return v.JSON(v.data)
	}
	_, err := ExecuteFunc(v.render.rootTemplateHTML, v.render.startTag, v.render.endTag, v.writer, func(w io.Writer, tag string) (int, error) {
		switch tag {
		case "view":
			return w.Write(s2b(view))
		case "data-page":
			pageJson, _ := json.Marshal(v.data)
			return HTMLEscape(w, pageJson)
		default:
		}
		return 0, nil
	})

	return err
}

func (v *View) JSON(data any) error {
	v.writer.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(v.writer).Encode(data)
}

func b2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func s2b(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

type TagFunc func(w io.Writer, tag string) (int, error)

// Use Template.ExecuteFunc for frozen templates.
func ExecuteFunc(template []byte, startTag, endTag string, w io.Writer, f TagFunc) (int64, error) {
	s := template
	a := s2b(startTag)
	b := s2b(endTag)

	var nn int64
	var ni int
	var err error
	for {
		n := bytes.Index(s, a)
		if n < 0 {
			break
		}
		ni, err = w.Write(s[:n])
		nn += int64(ni)
		if err != nil {
			return nn, err
		}

		s = s[n+len(a):]
		n = bytes.Index(s, b)
		if n < 0 {
			// cannot find end tag - just write it to the output.
			ni, _ = w.Write(a)
			nn += int64(ni)
			break
		}

		ni, err = f(w, b2s(s[:n]))
		nn += int64(ni)
		if err != nil {
			return nn, err
		}
		s = s[n+len(b):]
	}
	ni, err = w.Write(s)
	nn += int64(ni)

	return nn, err
}

// HTML escaping.

var (
	htmlQuot = []byte("&#34;") // shorter than "&quot;"
	htmlApos = []byte("&#39;") // shorter than "&apos;" and apos was not in HTML until HTML5
	htmlAmp  = []byte("&amp;")
	htmlLt   = []byte("&lt;")
	htmlGt   = []byte("&gt;")
	htmlNull = []byte("\uFFFD")
)

// HTMLEscape writes to w the escaped HTML equivalent of the plain text data b.
func HTMLEscape(w io.Writer, b []byte) (int, error) {
	last := 0
	n := 0
	var err error
	for i, c := range b {
		var html []byte
		switch c {
		case '\000':
			html = htmlNull
		case '"':
			html = htmlQuot
		case '\'':
			html = htmlApos
		case '&':
			html = htmlAmp
		case '<':
			html = htmlLt
		case '>':
			html = htmlGt
		default:
			continue
		}
		wn, err := w.Write(b[last:i])
		if err != nil {
			return n, err
		}
		n += wn
		wn, err = w.Write(html)
		if err != nil {
			return n, err
		}
		n += wn
		last = i + 1
	}
	wn, err := w.Write(b[last:])
	if err != nil {
		return n, err
	}
	n += wn
	return n, nil
}
