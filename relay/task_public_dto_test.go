package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/require"
)

func TestTaskModel2DtoRewritesSunoMediaURLs(t *testing.T) {
	previousServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://gateway.example"
	t.Cleanup(func() { system_setting.ServerAddress = previousServerAddress })

	data, err := common.Marshal([]dto.SunoSong{{
		AudioURL:      "https://provider.example/audio.mp3",
		VideoURL:      "https://provider.example/video.mp4",
		ImageURL:      "https://provider.example/image.jpg",
		ImageLargeURL: "https://provider.example/image-large.jpg",
	}})
	require.NoError(t, err)

	result := TaskModel2Dto(&model.Task{
		TaskID:   "task_public_1",
		Platform: constant.TaskPlatformSuno,
		Status:   model.TaskStatusSuccess,
		Data:     data,
	}, false)

	require.NotNil(t, result.Data)
	var songs []dto.SunoSong
	require.NoError(t, common.Unmarshal(result.Data, &songs))
	require.Len(t, songs, 1)
	require.NotContains(t, string(result.Data), "provider.example")
	require.Equal(t, "https://gateway.example/api/task/task_public_1/media?kind=audio&index=0", songs[0].AudioURL)
	require.Equal(t, "https://gateway.example/api/task/task_public_1/media?kind=video&index=0", songs[0].VideoURL)
	require.Equal(t, "https://gateway.example/api/task/task_public_1/media?kind=image&index=0", songs[0].ImageURL)
	require.Equal(t, "https://gateway.example/api/task/task_public_1/media?kind=image_large&index=0", songs[0].ImageLargeURL)
}
