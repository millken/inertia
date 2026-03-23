package web

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/millken/inertia/router"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

type HandlerFunc func(c *Context) error

type Server struct {
	router       *router.Router[HandlerFunc]
	middleware   []HandlerFunc
	contextPool  sync.Pool
	errorHandler func(*Context, error)
	ready        chan struct{}

	mu              sync.Mutex
	listener        net.Listener
	redirectServer  *http.Server
	activeConns     map[net.Conn]struct{}
	connWG          sync.WaitGroup
	shutdownCh      chan struct{}
	doneCh          chan struct{}
	shuttingDown    bool
	shutdownTimeout time.Duration
	readTimeout     time.Duration
	writeTimeout    time.Duration
	idleTimeout     time.Duration
	readBufferSize  int
	writeBufferSize int
	maxHeaderBytes  int
	maxBodyBytes    int64
}

func NewServer() *Server {
	server := &Server{
		router: router.New[HandlerFunc](),
		errorHandler: func(ctx *Context, err error) {
			slog.Error("web handler error", "path", ctx.Request.Path(), "error", err)
			if !ctx.Response.Written() {
				ctx.Response.WriteHeader(500)
			}
		},
		ready:           make(chan struct{}),
		activeConns:     make(map[net.Conn]struct{}),
		shutdownTimeout: 10 * time.Second,
		readBufferSize:  bufio.MaxScanTokenSize,
		writeBufferSize: 1024,
		maxHeaderBytes:  1 << 20,
	}
	server.contextPool.New = func() any {
		ctx := &serverContext{Context: Context{server: server}}
		ctx.Request.setReadBufferSize(server.currentReadBufferSize())
		ctx.Request.headers = make([]Header, 0, 16)
		ctx.Request.params = make(Params, 0, 8)
		ctx.Request.targetBuf = make([]byte, 0, 128)
		ctx.Request.headerBuf = make([]byte, 0, 256)
		ctx.Request.length = -1
		ctx.Response.body = make([]byte, 0, 1024)
		ctx.Response.headers = make([]Header, 0, 8)
		ctx.Response.status = 200
		ctx.Response.size = noWritten
		ctx.Response.writeBufferSize = server.currentWriteBufferSize()
		ctx.Request.setContext(context.Background())
		ctx.data = make(map[string]any, 8)
		return ctx
	}
	return server
}

type serverContext struct {
	Context
	tcpConn net.Conn
}

func (s *Server) GET(path string, handler HandlerFunc)     { s.router.Add("GET", path, handler) }
func (s *Server) POST(path string, handler HandlerFunc)    { s.router.Add("POST", path, handler) }
func (s *Server) PUT(path string, handler HandlerFunc)     { s.router.Add("PUT", path, handler) }
func (s *Server) DELETE(path string, handler HandlerFunc)  { s.router.Add("DELETE", path, handler) }
func (s *Server) PATCH(path string, handler HandlerFunc)   { s.router.Add("PATCH", path, handler) }
func (s *Server) OPTIONS(path string, handler HandlerFunc) { s.router.Add("OPTIONS", path, handler) }
func (s *Server) HEAD(path string, handler HandlerFunc)    { s.router.Add("HEAD", path, handler) }

func (s *Server) ANY(path string, handler HandlerFunc) {
	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"} {
		s.router.Add(method, path, handler)
	}
}

func (s *Server) Use(middleware ...HandlerFunc)                 { s.middleware = append(s.middleware, middleware...) }
func (s *Server) SetErrorHandler(handler func(*Context, error)) { s.errorHandler = handler }
func (s *Server) Ready() <-chan struct{}                        { return s.ready }
func (s *Server) SetShutdownTimeout(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdownTimeout = timeout
}
func (s *Server) SetReadTimeout(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readTimeout = timeout
}

func (s *Server) SetWriteTimeout(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeTimeout = timeout
}

func (s *Server) SetIdleTimeout(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idleTimeout = timeout
}

func (s *Server) SetMaxHeaderBytes(limit int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxHeaderBytes = limit
}

func (s *Server) SetReadBufferSize(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if size <= 0 {
		size = bufio.MaxScanTokenSize
	}
	s.readBufferSize = size
}

func (s *Server) SetWriteBufferSize(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if size <= 0 {
		size = 1024
	}
	s.writeBufferSize = size
}

func (s *Server) SetMaxBodyBytes(limit int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxBodyBytes = limit
}

func (s *Server) Request(method string, target string, headers []Header, body io.Reader) Response {
	ctx := s.acquireContext()
	defer s.releaseContext(ctx)
	s.requestIntoContext(ctx, method, target, headers, body)
	var response Response
	response.reset()
	response.status = ctx.Response.status
	response.size = ctx.Response.size
	response.written = ctx.Response.written
	response.body = bytes.Clone(ctx.Response.body)
	response.headers = append(response.headers[:0], ctx.Response.headers...)
	return response
}

func (s *Server) requestInto(method string, target string, headers []Header, body io.Reader, response *Response) {
	ctx := s.acquireContext()
	defer s.releaseContext(ctx)
	s.requestIntoContext(ctx, method, target, headers, body)
	response.stealFrom(&ctx.Response)
	ctx.Response.reset()
}

func (s *Server) requestIntoContext(ctx *serverContext, method string, target string, headers []Header, body io.Reader) {
	ctx.Request.setContext(context.Background())
	ctx.Request.setRemoteAddr("")
	ctx.Request.method = method
	ctx.Request.headers = append(ctx.Request.headers[:0], headers...)
	if body != nil {
		if bodyBytes, ok := body.(interface{ Bytes() []byte }); ok {
			ctx.Request.setOwnedBody(bodyBytes.Bytes())
		} else {
			payload, err := io.ReadAll(body)
			if err == nil {
				ctx.Request.setOwnedBody(payload)
			}
		}
	}
	scheme, host, path, query := ParseURL(target)
	ctx.Request.setURLParts(scheme, host, path, query)
	s.dispatch(&ctx.Context)
}

func (s *Server) Run(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	return s.serve(listener)
}

func (s *Server) RunTLS(address string, certFile string, keyFile string) error {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	listener, err := tls.Listen("tcp", address, &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
		NextProtos:   []string{"http/1.1"},
	})
	if err != nil {
		return err
	}
	return s.serve(listener)
}

func (s *Server) RunAutoTLS(address string, cacheDir string, hosts ...string) error {
	if len(hosts) == 0 {
		return errors.New("web: RunAutoTLS requires at least one host")
	}
	if cacheDir == "" {
		cacheDir = ".autocert"
	}
	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(cacheDir),
		HostPolicy: autocert.HostWhitelist(hosts...),
	}
	listener, err := tls.Listen("tcp", address, &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: manager.GetCertificate,
		NextProtos:     []string{"http/1.1", acme.ALPNProto},
	})
	if err != nil {
		return err
	}
	return s.serve(listener)
}

func (s *Server) RunAutoTLSRedirect(address string, httpAddress string, cacheDir string, hosts ...string) error {
	if httpAddress == "" {
		httpAddress = ":80"
	}
	if len(hosts) == 0 {
		return errors.New("web: RunAutoTLSRedirect requires at least one host")
	}
	if cacheDir == "" {
		cacheDir = ".autocert"
	}
	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(cacheDir),
		HostPolicy: autocert.HostWhitelist(hosts...),
	}
	redirectServer := startAutoHTTPRedirectServer(httpAddress, manager, httpsPort(address))
	listener, err := tls.Listen("tcp", address, &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: manager.GetCertificate,
		NextProtos:     []string{"http/1.1", acme.ALPNProto},
	})
	if err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = redirectServer.Shutdown(shutdownCtx)
		return err
	}
	return s.serve(listener, redirectServer)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if s.doneCh == nil {
		s.mu.Unlock()
		return nil
	}
	doneCh := s.doneCh
	if s.shuttingDown {
		s.mu.Unlock()
		select {
		case <-doneCh:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.shuttingDown = true
	listener := s.listener
	redirectServer := s.redirectServer
	connections := make([]net.Conn, 0, len(s.activeConns))
	for conn := range s.activeConns {
		connections = append(connections, conn)
	}
	shutdownCh := s.shutdownCh
	s.mu.Unlock()

	if shutdownCh != nil {
		close(shutdownCh)
	}
	if listener != nil {
		_ = listener.Close()
	}
	if redirectServer != nil {
		_ = redirectServer.Shutdown(ctx)
	}
	for _, conn := range connections {
		_ = conn.SetReadDeadline(time.Now())
	}

	waitCh := make(chan struct{})
	go func() {
		s.connWG.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		s.finishShutdown()
		return nil
	case <-ctx.Done():
		for _, conn := range connections {
			_ = conn.Close()
		}
		<-waitCh
		s.finishShutdown()
		return ctx.Err()
	}
}

func (s *Server) serve(listener net.Listener, redirectServer ...*http.Server) error {
	if err := s.beginServe(listener, redirectServer...); err != nil {
		return err
	}
	defer listener.Close()
	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				if s.isShuttingDown() || errors.Is(acceptErr, net.ErrClosed) {
					return
				}
				continue
			}
			go s.handleConnection(conn)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)
	close(s.ready)
	select {
	case <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), s.currentShutdownTimeout())
		defer cancel()
		return s.Shutdown(ctx)
	case <-s.doneChannel():
		return nil
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	s.trackConn(conn)
	defer s.untrackConn(conn)
	ctx := s.acquireContext()
	ctx.tcpConn = conn
	ctx.Request.reader.Reset(conn)
	ctx.Request.setContext(context.Background())
	ctx.Request.setRemoteAddr(conn.RemoteAddr().String())
	defer func() {
		conn.Close()
		s.releaseContext(ctx)
	}()
	firstRequest := true
	for {
		closeConn, ok := s.readRequest(ctx, firstRequest)
		if !ok {
			return
		}
		firstRequest = false
		s.dispatch(&ctx.Context)
		s.applyWriteDeadline(conn)
		if _, err := ctx.Response.WriteTo(conn); err != nil {
			return
		}
		s.clearWriteDeadline(conn)
		if closeConn || s.isShuttingDown() {
			return
		}
		ctx.Response.reset()
		ctx.routeHandler = nil
		ctx.nextIndex = 0
		clear(ctx.data)
		ctx.Request.params = ctx.Request.params[:0]
		ctx.Request.headers = ctx.Request.headers[:0]
		ctx.Request.body = ctx.Request.body[:0]
		ctx.Request.length = -1
		ctx.Request.consumed = 0
	}
}

func (s *Server) readRequest(ctx *serverContext, firstRequest bool) (bool, bool) {
	ctx.Response.reset()
	ctx.routeHandler = nil
	ctx.nextIndex = 0
	ctx.Request.params = ctx.Request.params[:0]
	ctx.Request.headers = ctx.Request.headers[:0]
	ctx.Request.body = ctx.Request.body[:0]
	ctx.Request.length = -1
	ctx.Request.consumed = 0
	ctx.Request.setReadBufferSize(s.currentReadBufferSize())
	s.applyReadDeadline(ctx.tcpConn, firstRequest)
	message, err := ctx.Request.reader.ReadSlice('\n')
	if err != nil {
		return false, false
	}
	headerBytes := len(message)
	if s.exceedsMaxHeaderBytes(headerBytes) {
		_, _ = io.WriteString(ctx.tcpConn, "HTTP/1.1 431 Request Header Fields Too Large\r\nConnection: close\r\n\r\n")
		return false, false
	}
	methodBytes, targetBytes, ok := parseRequestLineBytes(message)
	if !ok {
		_, _ = io.WriteString(ctx.tcpConn, "HTTP/1.1 400 Bad Request\r\n\r\n")
		return false, false
	}
	s.applyRequestReadDeadline(ctx.tcpConn)
	method, ok := methodString(methodBytes)
	if !ok || !isRequestMethod(method) {
		_, _ = io.WriteString(ctx.tcpConn, "HTTP/1.1 400 Bad Request\r\n\r\n")
		return false, false
	}
	ctx.Request.method = method
	scheme, host, path, query := parseTargetBytes(targetBytes)
	ctx.Request.setURLPartsBytes(scheme, host, path, query)
	closeConn := false
	for {
		message, err = ctx.Request.reader.ReadSlice('\n')
		if err != nil {
			return false, false
		}
		headerBytes += len(message)
		if s.exceedsMaxHeaderBytes(headerBytes) {
			_, _ = io.WriteString(ctx.tcpConn, "HTTP/1.1 431 Request Header Fields Too Large\r\nConnection: close\r\n\r\n")
			return false, false
		}
		if isHeaderTerminator(message) {
			break
		}
		key, value, ok := parseHeaderBytes(message)
		if !ok {
			continue
		}
		ctx.Request.appendHeaderBytes(key, value)
		if bytes.EqualFold(key, s2b("Connection")) && bytes.EqualFold(value, s2b("close")) {
			closeConn = true
		}
	}
	ctx.Request.setRemoteAddr(ctx.tcpConn.RemoteAddr().String())
	if ctx.Request.host == "" {
		ctx.Request.host = ctx.Request.Header("Host")
	}
	contentLength := ctx.Request.ensureContentLength()
	if s.exceedsMaxBodyBytes(contentLength) {
		_, _ = io.WriteString(ctx.tcpConn, "HTTP/1.1 413 Payload Too Large\r\nConnection: close\r\n\r\n")
		return false, false
	}
	if contentLength > 0 {
		payload := ctx.Request.prepareBody(ctx.Request.length)
		if _, err := io.ReadFull(&ctx.Request.reader, payload); err != nil {
			return false, false
		}
		ctx.Request.bindBodyReader()
	}
	s.clearReadDeadline(ctx.tcpConn)
	return closeConn, true
}

func (s *Server) dispatch(ctx *Context) {
	found, handler := s.router.LookupNoAlloc(ctx.Request.method, ctx.Request.path, ctx.Request.addParameter)
	if found {
		ctx.routeHandler = handler
	} else {
		ctx.Response.WriteHeader(404)
	}
	if err := ctx.Next(); err != nil && s.errorHandler != nil {
		s.errorHandler(ctx, err)
	}
}

func (s *Server) acquireContext() *serverContext {
	ctx := s.contextPool.Get().(*serverContext)
	ctx.server = s
	ctx.Request.reset()
	ctx.Request.setReadBufferSize(s.currentReadBufferSize())
	ctx.Response.reset()
	ctx.Response.writeBufferSize = s.currentWriteBufferSize()
	ctx.Request.setContext(context.Background())
	clear(ctx.data)
	return ctx
}

func (s *Server) releaseContext(ctx *serverContext) {
	ctx.Request.reset()
	ctx.Response.reset()
	ctx.routeHandler = nil
	ctx.nextIndex = 0
	ctx.tcpConn = nil
	clear(ctx.data)
	s.contextPool.Put(ctx)
}

func isRequestMethod(method string) bool {
	switch method {
	case "GET", "HEAD", "POST", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH":
		return true
	default:
		return false
	}
}

func parseRequestLineBytes(line []byte) (method []byte, target []byte, ok bool) {
	end := trimLineEnd(line)
	if end == 0 {
		return nil, nil, false
	}
	space := bytes.IndexByte(line[:end], ' ')
	if space <= 0 {
		return nil, nil, false
	}
	lastSpace := bytes.LastIndexByte(line[:end], ' ')
	if lastSpace == space {
		lastSpace = end
	}
	space++
	if space > lastSpace {
		return nil, nil, false
	}
	return line[:space-1], line[space:lastSpace], true
}

func parseHeaderBytes(line []byte) (key []byte, value []byte, ok bool) {
	end := trimLineEnd(line)
	if end == 0 {
		return nil, nil, false
	}
	colon := bytes.IndexByte(line[:end], ':')
	if colon <= 0 || colon >= end {
		return nil, nil, false
	}
	valueStart := colon + 1
	if valueStart < end && line[valueStart] == ' ' {
		valueStart++
	}
	return line[:colon], line[valueStart:end], true
}

func isHeaderTerminator(line []byte) bool {
	return len(line) == 2 && line[0] == '\r' && line[1] == '\n'
}

func trimLineEnd(line []byte) int {
	end := len(line)
	if end > 0 && line[end-1] == '\n' {
		end--
	}
	if end > 0 && line[end-1] == '\r' {
		end--
	}
	return end
}

func methodString(method []byte) (string, bool) {
	switch b2s(method) {
	case "GET":
		return "GET", true
	case "HEAD":
		return "HEAD", true
	case "POST":
		return "POST", true
	case "PUT":
		return "PUT", true
	case "DELETE":
		return "DELETE", true
	case "CONNECT":
		return "CONNECT", true
	case "OPTIONS":
		return "OPTIONS", true
	case "TRACE":
		return "TRACE", true
	case "PATCH":
		return "PATCH", true
	default:
		return "", false
	}
}

func parseTargetBytes(target []byte) (scheme []byte, host []byte, path []byte, query []byte) {
	schemePos := bytes.Index(target, []byte("://"))
	if schemePos != -1 {
		scheme = target[:schemePos]
		target = target[schemePos+3:]
	}
	pathPos := bytes.IndexByte(target, '/')
	if pathPos != -1 {
		host = target[:pathPos]
		target = target[pathPos:]
	}
	queryPos := bytes.IndexByte(target, '?')
	if queryPos != -1 {
		path = target[:queryPos]
		query = target[queryPos+1:]
		return
	}
	path = target
	return
}

func startAutoHTTPRedirectServer(address string, manager *autocert.Manager, targetPort string) *http.Server {
	redirectHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		target := buildHTTPSRedirectURL(request.Host, request.URL.RequestURI(), targetPort)
		http.Redirect(writer, request, target, http.StatusMovedPermanently)
	})
	server := &http.Server{
		Addr:    address,
		Handler: manager.HTTPHandler(redirectHandler),
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("web autossl redirect server error", "addr", address, "error", err)
		}
	}()
	return server
}

func (s *Server) beginServe(listener net.Listener, redirectServer ...*http.Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return errors.New("web: server is already running")
	}
	s.listener = listener
	if len(redirectServer) > 0 {
		s.redirectServer = redirectServer[0]
	} else {
		s.redirectServer = nil
	}
	s.shutdownCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.shuttingDown = false
	return nil
}

func (s *Server) currentShutdownTimeout() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shutdownTimeout <= 0 {
		return 10 * time.Second
	}
	return s.shutdownTimeout
}

func (s *Server) currentReadTimeout() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readTimeout
}

func (s *Server) currentWriteTimeout() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeTimeout
}

func (s *Server) currentIdleTimeout() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.idleTimeout
}

func (s *Server) currentMaxHeaderBytes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxHeaderBytes
}

func (s *Server) currentReadBufferSize() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.readBufferSize <= 0 {
		return bufio.MaxScanTokenSize
	}
	return s.readBufferSize
}

func (s *Server) currentWriteBufferSize() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writeBufferSize <= 0 {
		return 1024
	}
	return s.writeBufferSize
}

func (s *Server) currentMaxBodyBytes() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxBodyBytes
}

func (s *Server) doneChannel() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.doneCh
}

func (s *Server) finishShutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doneCh != nil {
		close(s.doneCh)
	}
	s.listener = nil
	s.redirectServer = nil
	s.shutdownCh = nil
	s.doneCh = nil
	s.shuttingDown = false
	for conn := range s.activeConns {
		delete(s.activeConns, conn)
	}
}

func (s *Server) trackConn(conn net.Conn) {
	s.mu.Lock()
	s.activeConns[conn] = struct{}{}
	s.connWG.Add(1)
	s.mu.Unlock()
}

func (s *Server) untrackConn(conn net.Conn) {
	s.mu.Lock()
	delete(s.activeConns, conn)
	s.mu.Unlock()
	s.connWG.Done()
}

func (s *Server) isShuttingDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shuttingDown
}

func (s *Server) applyReadDeadline(conn net.Conn, firstRequest bool) {
	if conn == nil {
		return
	}
	deadline := s.currentReadTimeout()
	if !firstRequest {
		if idleTimeout := s.currentIdleTimeout(); idleTimeout > 0 {
			deadline = idleTimeout
		}
	}
	if deadline > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(deadline))
		return
	}
	_ = conn.SetReadDeadline(time.Time{})
}

func (s *Server) applyRequestReadDeadline(conn net.Conn) {
	if conn == nil {
		return
	}
	deadline := s.currentReadTimeout()
	if deadline > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(deadline))
	}
}

func (s *Server) clearReadDeadline(conn net.Conn) {
	if conn == nil {
		return
	}
	_ = conn.SetReadDeadline(time.Time{})
}

func (s *Server) applyWriteDeadline(conn net.Conn) {
	if conn == nil {
		return
	}
	deadline := s.currentWriteTimeout()
	if deadline > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(deadline))
		return
	}
	_ = conn.SetWriteDeadline(time.Time{})
}

func (s *Server) clearWriteDeadline(conn net.Conn) {
	if conn == nil {
		return
	}
	_ = conn.SetWriteDeadline(time.Time{})
}

func (s *Server) exceedsMaxHeaderBytes(size int) bool {
	limit := s.currentMaxHeaderBytes()
	return limit > 0 && size > limit
}

func (s *Server) exceedsMaxBodyBytes(size int) bool {
	limit := s.currentMaxBodyBytes()
	return limit > 0 && int64(size) > limit
}

func httpsPort(address string) string {
	_, port, err := net.SplitHostPort(address)
	if err == nil && port != "" {
		return port
	}
	if strings.HasPrefix(address, ":") {
		return strings.TrimPrefix(address, ":")
	}
	return "443"
}

func buildHTTPSRedirectURL(host string, requestURI string, targetPort string) string {
	if requestURI == "" {
		requestURI = "/"
	}
	redirectHost := normalizeHTTPSHost(host, targetPort)
	return fmt.Sprintf("https://%s%s", redirectHost, requestURI)
}

func normalizeHTTPSHost(host string, targetPort string) string {
	if host == "" {
		if targetPort == "" || targetPort == "443" {
			return "localhost"
		}
		return net.JoinHostPort("localhost", targetPort)
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	if targetPort == "" || targetPort == "443" {
		return host
	}
	return net.JoinHostPort(host, targetPort)
}
