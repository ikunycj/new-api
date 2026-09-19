package controller

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// contextKeyInternalErrorDetail marks a request whose caller is allowed to see
// the gateway's internal error classification in the response headers.
//
// It is set only after a managed load-test run has been authenticated (see the
// signature check in Relay), so it cannot be claimed by a client.
const contextKeyInternalErrorDetail = "internal_error_detail"

// internalErrorDetailAllowed reports whether the internal error headers may be
// written for this request.
//
// In passthrough mode nothing changes, so they are always written. Once the
// projection is enabled, only authenticated internal callers keep them: the
// managed load-test agent reads X-Alltoken-Code to classify failures in its own
// report, while a normal caller has no use for the channel-scoped identifiers
// those headers carry.
func internalErrorDetailAllowed(c *gin.Context) bool {
	if !common.IsPublicErrorNormalized() {
		return true
	}
	return c != nil && c.GetBool(contextKeyInternalErrorDetail)
}

// writeRelayError renders the client-facing error for a failed relay request.
//
// Two things happen here that must not be conflated:
//
//   - Projection. In normalized mode the body carries none of the internal
//     classification and none of the upstream's own wording. That decision
//     lives in types.PublicError*; this function only writes it.
//   - Transport. If the response body has already started (a streaming request
//     that failed mid-stream), the error has to be delivered in-band. Appending
//     a JSON body to a half-written SSE stream produces a payload no client can
//     parse, so the failure surfaces to the caller as a decode error instead of
//     the actual condition.
func writeRelayError(c *gin.Context, relayFormat types.RelayFormat, ws *websocket.Conn, apiErr *types.NewAPIError, requestId string) {
	if apiErr == nil {
		return
	}

	opts := types.PublicErrorOptions{MessageSuffix: fmt.Sprintf("request id: %s", requestId)}

	if c.Writer != nil && c.Writer.Size() > 0 {
		writeStreamedError(c, relayFormat, ws, apiErr, opts)
		return
	}

	switch relayFormat {
	case types.RelayFormatOpenAIRealtime:
		helper.WssError(c, ws, apiErr.ToPublicOpenAIError(opts))
	case types.RelayFormatClaude:
		c.JSON(apiErr.PublicStatusCode(opts), gin.H{
			"type":  "error",
			"error": apiErr.ToPublicClaudeError(opts),
		})
	default:
		c.JSON(apiErr.PublicStatusCode(opts), gin.H{
			"error": apiErr.ToPublicOpenAIError(opts),
		})
	}
}

// writeStreamedError delivers an error for a response that has already begun.
//
// The HTTP status was committed by the first chunk, so the failure is reported
// as a terminal event of the stream itself, matching the format the caller
// asked for.
func writeStreamedError(c *gin.Context, relayFormat types.RelayFormat, ws *websocket.Conn, apiErr *types.NewAPIError, opts types.PublicErrorOptions) {
	switch relayFormat {
	case types.RelayFormatOpenAIRealtime:
		helper.WssError(c, ws, apiErr.ToPublicOpenAIError(opts))
		return
	case types.RelayFormatClaude:
		payload, err := common.Marshal(gin.H{
			"type":  "error",
			"error": apiErr.ToPublicClaudeError(opts),
		})
		if err != nil {
			return
		}
		_, _ = c.Writer.Write([]byte("event: error\ndata: " + string(payload) + "\n\n"))
	default:
		payload, err := common.Marshal(gin.H{"error": apiErr.ToPublicOpenAIError(opts)})
		if err != nil {
			return
		}
		_, _ = c.Writer.Write([]byte("data: " + string(payload) + "\n\n"))
	}
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
	logger.LogError(c, fmt.Sprintf("relay stream error after response started: %s", common.LocalLogPreview(apiErr.Error())))
}
