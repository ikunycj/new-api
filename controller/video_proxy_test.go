package controller

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVideoProxyTestDB(t *testing.T) {
	t.Helper()
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.Task{}, &model.Channel{}))

	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	previousCacheEnabled := common.MemoryCacheEnabled
	model.DB, model.LOG_DB = testDB, testDB
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		common.MemoryCacheEnabled = previousCacheEnabled
	})
}

func TestVideoProxyRejectsTaskOwnedByAnotherUser(t *testing.T) {
	setupVideoProxyTestDB(t)
	require.NoError(t, model.DB.Create(&model.Task{
		TaskID:    "task_owned_by_user_7",
		UserId:    7,
		ChannelId: 23,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://private-supplier.example/video.mp4",
		},
	}).Error)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/videos/:task_id/content", func(c *gin.Context) {
		c.Set("id", 8)
		VideoProxy(c)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/videos/task_owned_by_user_7/content", nil))

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Task not found")
	assert.NotContains(t, recorder.Body.String(), "private-supplier")
	assert.NotContains(t, recorder.Body.String(), "video.mp4")
}

func TestVideoProxyReturnsStoredMediaThroughGateway(t *testing.T) {
	setupVideoProxyTestDB(t)
	videoPayload := base64.StdEncoding.EncodeToString([]byte("video-bytes"))
	require.NoError(t, model.DB.Create(&model.Channel{Id: 23, Name: "private-supplier", Key: "secret-key"}).Error)
	require.NoError(t, model.DB.Create(&model.Task{
		TaskID:    "task_video",
		UserId:    7,
		ChannelId: 23,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			ResultURL: "data:video/mp4;base64," + videoPayload,
		},
	}).Error)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/videos/:task_id/content", func(c *gin.Context) {
		c.Set("id", 7)
		VideoProxy(c)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/videos/task_video/content", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "video/mp4", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "video-bytes", recorder.Body.String())
	assert.NotContains(t, recorder.Body.String(), "private-supplier")
}
