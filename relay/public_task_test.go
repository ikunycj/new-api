package relay

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicTaskViewKeepsSuccessfulLegacyResultAndInternalIdentity(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public", ChannelId: 23, Status: model.TaskStatusSuccess, Quota: 100,
		FailReason: "https://cdn.example.com/result.mp4",
		Properties: model.Properties{Input: "my prompt", OriginModelName: "video-model", UpstreamModelName: "private-model"},
		Data:       json.RawMessage(`{"channel_id":23,"channel_name":"supplier","result":{"url":"https://cdn.example.com/result.mp4"}}`),
	}
	view := TaskModel2PublicDto(task)
	assert.Equal(t, 1, view.ChannelId)
	assert.Equal(t, string(model.TaskStatusSuccess), view.Status)
	assert.Equal(t, "http://localhost:3000/v1/videos/task_public/content", view.ResultURL)
	assert.Empty(t, view.FailReason)
	assert.Equal(t, 100, view.Quota)
	encoded, err := common.Marshal(view)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "private-model")
	assert.NotContains(t, string(encoded), "supplier")
	assert.Contains(t, string(encoded), "my prompt")
	assert.Equal(t, 23, task.ChannelId)
	assert.Equal(t, "private-model", task.Properties.UpstreamModelName)
	assert.Equal(t, task.Data, TaskModel2Dto(task).Data)
}

func TestPublicTaskFailureDoesNotExposeReasonAsResultURL(t *testing.T) {
	task := &model.Task{ChannelId: 23, Status: model.TaskStatusFailure, FailReason: "private-supplier pool exhausted"}
	view := TaskModel2PublicDto(task)
	assert.Empty(t, view.ResultURL)
	assert.Equal(t, common.DefaultPublicChannelErrorMessage(), view.FailReason)
	assert.Equal(t, task.FailReason, TaskModel2Dto(task).FailReason)
}
