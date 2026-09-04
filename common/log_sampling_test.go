package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldLogAccessRequestAlwaysKeepsFailuresAndSlowRequests(t *testing.T) {
	SetAccessLogSuccessSampleRate(0)
	t.Cleanup(func() { SetAccessLogSuccessSampleRate(1) })
	require.True(t, ShouldLogAccessRequest(500, 1, "request-1", "/v1/chat/completions"))
	require.True(t, ShouldLogAccessRequest(200, 1000, "request-2", "/v1/chat/completions"))
}

func TestShouldLogAccessRequestHonorsSampleRateBounds(t *testing.T) {
	SetAccessLogSuccessSampleRate(0)
	require.False(t, ShouldLogAccessRequest(200, 1, "request-1", "/v1/chat/completions"))
	SetAccessLogSuccessSampleRate(1)
	require.True(t, ShouldLogAccessRequest(200, 1, "request-1", "/v1/chat/completions"))
	SetAccessLogSuccessSampleRate(-1)
	require.Equal(t, float64(0), GetAccessLogSuccessSampleRate())
	SetAccessLogSuccessSampleRate(2)
	require.Equal(t, float64(1), GetAccessLogSuccessSampleRate())
}
