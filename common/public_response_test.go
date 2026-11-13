package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPublicChannelErrorUsesCurrentContact(t *testing.T) {
	OptionMapRWMutex.Lock()
	previous := OptionMap
	OptionMap = map[string]string{"CustomerService": "客服邮箱：support@example.com\n服务时间：09:00–18:00"}
	OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		OptionMapRWMutex.Lock()
		OptionMap = previous
		OptionMapRWMutex.Unlock()
	})
	assert.Equal(t, "服务暂异常，大概率是临时错误，请重试，如多次重试无效，请联系管理员\n客服邮箱：support@example.com\n服务时间：09:00–18:00", DefaultPublicChannelErrorMessage())
	OptionMapRWMutex.Lock()
	OptionMap["CustomerService"] = " "
	OptionMapRWMutex.Unlock()
	assert.Equal(t, "服务暂异常，大概率是临时错误，请重试，如多次重试无效，请联系管理员", DefaultPublicChannelErrorMessage())
}

func TestPublicTaskJSONHidesDiagnosticsButPreservesResultAndInput(t *testing.T) {
	original := json.RawMessage(`{"channel_id":23,"channel_name":"private-supplier","task_id":"task_public","created_at":9007199254740993,"data":[{"url":"https://cdn.example.com/result.mp4","metadata":{"error_message":"supplier account exhausted","upstream_model_name":"private-model"}}],"input":{"channel_name":"user text"},"error":{"code":"pool_exhausted","message":"private-supplier","cause":{"channel_id":23}}}`)
	before := string(original)
	projected, err := PublicTaskJSON(original)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, Unmarshal(projected, &fields))
	assert.Equal(t, "1", string(fields["channel_id"]))
	assert.Equal(t, `"auto"`, string(fields["channel_name"]))
	assert.Equal(t, "9007199254740993", string(fields["created_at"]))
	assert.JSONEq(t, `{"channel_name":"user text"}`, string(fields["input"]))
	assert.Contains(t, string(projected), "https://cdn.example.com/result.mp4")
	assert.Contains(t, string(projected), "task_public")
	assert.NotContains(t, string(projected), "private-supplier")
	assert.NotContains(t, string(projected), "supplier account exhausted")
	assert.NotContains(t, string(projected), "private-model")
	assert.NotContains(t, string(projected), "pool_exhausted")
	assert.NotContains(t, string(projected), "cause")
	assert.Equal(t, before, string(original))
}

func TestPublicTaskJSONRejectsMalformedMetadata(t *testing.T) {
	_, err := PublicTaskJSON(json.RawMessage(`{"channel_id":23`))
	require.Error(t, err)
}

func TestPublicSunoDataRewritesClipMediaURLs(t *testing.T) {
	data := json.RawMessage(`[{"id":"clip-1","audio_url":"https://provider.example/audio.mp3","image_url":"https://provider.example/image.png"}]`)
	projected, err := PublicSunoData("task_public", data)
	require.NoError(t, err)
	assert.Contains(t, string(projected), "/suno/media/task_public/clip-1/audio")
	assert.Contains(t, string(projected), "/suno/media/task_public/clip-1/image")
	assert.NotContains(t, string(projected), "provider.example")
}
