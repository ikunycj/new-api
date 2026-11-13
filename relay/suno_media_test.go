package relay

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSunoMediaTestDB(t *testing.T) {
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
	model.InitChannelCache()
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		common.MemoryCacheEnabled = previousCacheEnabled
	})
}

func configureSunoMediaHTTPTest(t *testing.T) {
	t.Helper()
	fetchSetting := system_setting.GetFetchSetting()
	previous := *fetchSetting
	fetchSetting.EnableSSRFProtection = true
	fetchSetting.AllowPrivateIp = true
	fetchSetting.DomainFilterMode = false
	fetchSetting.IpFilterMode = false
	fetchSetting.DomainList = nil
	fetchSetting.IpList = nil
	fetchSetting.AllowedPorts = nil
	fetchSetting.ApplyIPFilterForDomain = true
	t.Cleanup(func() {
		*fetchSetting = previous
	})
	service.InitHttpClient()
}

func newSunoMediaRouter(userID int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/suno/:task_id/:clip_id/:kind", func(c *gin.Context) {
		c.Set("id", userID)
		RelaySunoMedia(c)
	})
	return router
}

func TestRelaySunoMediaProxiesMediaWithoutProviderURL(t *testing.T) {
	setupSunoMediaTestDB(t)
	configureSunoMediaHTTPTest(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("audio-bytes"))
	}))
	t.Cleanup(upstream.Close)

	data, err := common.Marshal([]map[string]string{{
		"id":        "clip-1",
		"audio_url": upstream.URL + "/audio.mp3",
	}})
	require.NoError(t, err)
	require.NoError(t, model.DB.Create(&model.Channel{Id: 23, Name: "private-supplier", Key: "secret-key"}).Error)
	require.NoError(t, model.DB.Create(&model.Task{
		TaskID:    "task_suno",
		Platform:  constant.TaskPlatformSuno,
		UserId:    7,
		ChannelId: 23,
		Data:      data,
		Status:    model.TaskStatusSuccess,
	}).Error)

	recorder := httptest.NewRecorder()
	newSunoMediaRouter(7).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/suno/task_suno/clip-1/audio", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "audio/mpeg", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "audio-bytes", recorder.Body.String())
	assert.NotContains(t, recorder.Body.String(), upstream.URL)
}

func TestRelaySunoMediaHidesUpstreamErrorBody(t *testing.T) {
	setupSunoMediaTestDB(t)
	configureSunoMediaHTTPTest(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"private supplier details"}`))
	}))
	t.Cleanup(upstream.Close)

	data, err := common.Marshal([]map[string]string{{
		"id":        "clip-1",
		"audio_url": upstream.URL + "/audio.mp3",
	}})
	require.NoError(t, err)
	require.NoError(t, model.DB.Create(&model.Channel{Id: 23, Name: "private-supplier", Key: "secret-key"}).Error)
	require.NoError(t, model.DB.Create(&model.Task{
		TaskID:    "task_suno",
		Platform:  constant.TaskPlatformSuno,
		UserId:    7,
		ChannelId: 23,
		Data:      data,
		Status:    model.TaskStatusSuccess,
	}).Error)

	recorder := httptest.NewRecorder()
	newSunoMediaRouter(7).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/suno/task_suno/clip-1/audio", nil))

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Contains(t, recorder.Body.String(), common.DefaultPublicChannelErrorMessage())
	assert.NotContains(t, recorder.Body.String(), "private supplier details")
	assert.NotContains(t, recorder.Body.String(), upstream.URL)
}
