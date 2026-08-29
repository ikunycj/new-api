package service

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const mockLoadTestSignatureVersion = "v1"

// MockLoadTestSignature binds the managed run to its API token and exact mock
// configuration. It is deliberately derived from the server CryptoSecret so
// it is valid across application instances while never exposing a channel key
// or URL to the load-test agent.
func MockLoadTestSignature(runID string, tokenID int, channelsJSON string, failureRate float64, failureStatus, latencyMS int) string {
	data := fmt.Sprintf("%s:%s:%d:%s:%g:%d:%d", mockLoadTestSignatureVersion, strings.TrimSpace(runID), tokenID, channelsJSON, failureRate, failureStatus, latencyMS)
	return common.GenerateHMAC(data)
}

// VerifyMockLoadTestRequest authenticates the internal mock executor marker.
// A malformed or unauthenticated marker is rejected before pricing, channel
// selection, pre-consumption, or any upstream HTTP call can happen.
func VerifyMockLoadTestRequest(c *gin.Context, channelsJSON string, failureRate float64, failureStatus, latencyMS int) error {
	if c == nil || c.Request == nil || !strings.EqualFold(strings.TrimSpace(c.GetHeader(constant.MockLoadTestHeader)), "true") {
		return nil
	}
	runID := strings.TrimSpace(c.GetHeader(constant.MockLoadTestRunHeader))
	if runID == "" || len(runID) > 64 {
		return types.NewErrorWithStatusCode(fmt.Errorf("mock load-test run id is missing"), types.ErrorCodeAccessDenied, http.StatusForbidden, types.ErrOptionWithSkipRetry())
	}
	tokenID := c.GetInt("token_id")
	if tokenID <= 0 {
		return types.NewErrorWithStatusCode(fmt.Errorf("mock load-test token context is missing"), types.ErrorCodeAccessDenied, http.StatusForbidden, types.ErrOptionWithSkipRetry())
	}
	provided := strings.TrimSpace(c.GetHeader(constant.MockLoadTestTokenHeader))
	for _, signedRunID := range mockLoadTestSignedRunIDs(runID) {
		expected := MockLoadTestSignature(signedRunID, tokenID, channelsJSON, failureRate, failureStatus, latencyMS)
		if len(provided) == len(expected) && subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1 {
			return nil
		}
	}
	return types.NewErrorWithStatusCode(fmt.Errorf("mock load-test authorization is invalid"), types.ErrorCodeAccessDenied, http.StatusForbidden, types.ErrOptionWithSkipRetry())
}

// mockLoadTestSignedRunIDs accepts the exact server-issued run ID and the
// legacy Agent trace form (<run-id>-<vu>-<iteration>). The latter keeps older
// managed agents compatible while the per-request trace remains unique.
func mockLoadTestSignedRunIDs(runID string) []string {
	ids := []string{runID}
	lastDash := strings.LastIndexByte(runID, '-')
	if lastDash <= 0 || lastDash == len(runID)-1 {
		return ids
	}
	previousDash := strings.LastIndexByte(runID[:lastDash], '-')
	if previousDash <= 0 || previousDash == lastDash-1 {
		return ids
	}
	if _, err := strconv.Atoi(runID[lastDash+1:]); err != nil || strings.IndexFunc(runID[lastDash+1:], func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return ids
	}
	if _, err := strconv.Atoi(runID[previousDash+1 : lastDash]); err != nil || strings.IndexFunc(runID[previousDash+1:lastDash], func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return ids
	}
	base := runID[:previousDash]
	if base != "" {
		ids = append(ids, base)
	}
	return ids
}
