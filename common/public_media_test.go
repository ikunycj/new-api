package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicVideoDataRewritesProviderVideoURL(t *testing.T) {
	data := json.RawMessage(`{"result":{"video_url":"https://provider.example/video.mp4"}}`)
	projected, err := PublicVideoData("task_public", data)
	require.NoError(t, err)
	assert.Contains(t, string(projected), "/v1/videos/task_public/content")
	assert.NotContains(t, string(projected), "provider.example")
}
