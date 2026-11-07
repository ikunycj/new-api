package relay

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeMidjourneyResponseBodyHidesMediaOrigins(t *testing.T) {
	body := []byte(`{"code":1,"properties":{"imageUrl":"https://provider.example/image.png","videoUrls":[{"url":"https://provider.example/video.mp4"}]}}`)

	sanitized, err := sanitizeMidjourneyResponseBody(body, "task-123")
	require.NoError(t, err)
	require.NotContains(t, string(sanitized), "provider.example")
	require.Contains(t, string(sanitized), "/mj/image/task-123")
	require.Contains(t, string(sanitized), "/mj/video/task-123?index=0")
}
