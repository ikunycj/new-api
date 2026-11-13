package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/gin-gonic/gin"
)

func CloseResponseBodyGracefully(httpResponse *http.Response) {
	if httpResponse == nil || httpResponse.Body == nil {
		return
	}
	err := httpResponse.Body.Close()
	if err != nil {
		common.SysError("failed to close response body: " + err.Error())
	}
}

// ShouldCopyUpstreamHeader permits protocol headers, not supplier diagnostics.
// Gateway-owned CORS, authentication and request IDs are set independently.
func ShouldCopyUpstreamHeader(c *gin.Context, k string, v []string) bool {
	name := strings.ToLower(k)
	if strings.EqualFold(k, common.RequestIdKey) || name == "x-request-id" || name == "request-id" || name == "x-upstream-request-id" {
		if c != nil && len(v) > 0 {
			c.Set(common.UpstreamRequestIdKey, v[0])
		}
		return false
	}
	switch name {
	case "content-type", "content-disposition", "content-encoding", "content-range", "accept-ranges",
		"cache-control", "expires", "last-modified", "x-reasoning-included", "x-codex-turn-state":
		return true
	case "retry-after":
		if len(v) != 1 {
			return false
		}
		value := strings.TrimSpace(v[0])
		if _, err := strconv.ParseUint(value, 10, 32); err == nil {
			return true
		}
		_, err := http.ParseTime(value)
		return err == nil
	default:
		return false
	}
}

func IOCopyBytesGracefully(c *gin.Context, src *http.Response, data []byte) {
	if c.Writer == nil {
		return
	}

	body := io.NopCloser(bytes.NewBuffer(data))

	// We shouldn't set the header before we parse the response body, because the parse part may fail.
	// And then we will have to send an error response, but in this case, the header has already been set.
	// So the httpClient will be confused by the response.
	// For example, Postman will report error, and we cannot check the response at all.
	if src != nil {
		for k, v := range src.Header {
			if !ShouldCopyUpstreamHeader(c, k, v) {
				continue
			}
			c.Writer.Header().Set(k, v[0])
		}
	}

	// set Content-Length header manually BEFORE calling WriteHeader
	c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))

	// Write header with status code (this sends the headers)
	if src != nil {
		c.Writer.WriteHeader(src.StatusCode)
	} else {
		c.Writer.WriteHeader(http.StatusOK)
	}

	_, err := io.Copy(c.Writer, body)
	if err != nil {
		logger.LogError(c, fmt.Sprintf("failed to copy response body: %s", err.Error()))
	}
	c.Writer.Flush()
}
