package middleware

import (
	"bufio"
	"net"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// PublicRelayResponse removes diagnostic headers at the commit boundary. It
// never buffers or rewrites response bodies.
func PublicRelayResponse() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer = &publicResponseWriter{ResponseWriter: c.Writer}
		c.Next()
	}
}

type publicResponseWriter struct {
	gin.ResponseWriter
}

func (w *publicResponseWriter) filterHeaders() {
	for name := range w.Header() {
		key := strings.ToLower(name)
		if strings.HasPrefix(key, "x-error-") || strings.HasPrefix(key, "x-alltoken-") ||
			strings.HasPrefix(key, "x-channel-") || key == strings.ToLower(common.UpstreamRequestIdKey) ||
			key == "server" || key == "via" || key == "location" {
			w.Header().Del(name)
		}
	}
}

func (w *publicResponseWriter) Write(data []byte) (int, error) {
	w.filterHeaders()
	return w.ResponseWriter.Write(data)
}

func (w *publicResponseWriter) WriteString(data string) (int, error) {
	w.filterHeaders()
	return w.ResponseWriter.WriteString(data)
}

func (w *publicResponseWriter) WriteHeader(status int) {
	w.filterHeaders()
	w.ResponseWriter.WriteHeader(status)
}

func (w *publicResponseWriter) WriteHeaderNow() {
	w.filterHeaders()
	w.ResponseWriter.WriteHeaderNow()
}

func (w *publicResponseWriter) Flush() {
	w.filterHeaders()
	w.ResponseWriter.Flush()
}

func (w *publicResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.filterHeaders()
	return w.ResponseWriter.Hijack()
}

func (w *publicResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
