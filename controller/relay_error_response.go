package controller

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// writeRelayError projects internal failures at the downstream boundary. Only
// the local request ID is included so administrators can trace a support report.
func writeRelayError(c *gin.Context, relayFormat types.RelayFormat, ws *websocket.Conn, apiErr *types.NewAPIError, requestID string) {
	if apiErr == nil {
		return
	}
	opts := types.PublicErrorOptions{}
	if requestID != "" {
		opts.MessageSuffix = fmt.Sprintf("request id: %s", requestID)
	}
	if relayFormat == types.RelayFormatOpenAIRealtime {
		helper.WssError(c, ws, apiErr.ToPublicOpenAIError(opts))
		return
	}
	var payload gin.H
	if relayFormat == types.RelayFormatClaude {
		payload = gin.H{"type": "error", "error": apiErr.ToPublicClaudeError(opts)}
	} else {
		payload = gin.H{"error": apiErr.ToPublicOpenAIError(opts)}
	}
	if !c.Writer.Written() {
		c.JSON(apiErr.PublicStatusCode(), payload)
		return
	}
	data, err := common.Marshal(payload)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if relayFormat == types.RelayFormatClaude {
		_, _ = fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", data)
	} else {
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	}
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}
