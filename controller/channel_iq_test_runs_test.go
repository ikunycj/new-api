package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIQHistoryFiltersAndOrdersAcrossEngines(t *testing.T) {
	db, channel, _, _, _ := setupChannelRecoveryControllerDB(t)
	runs := []model.ChannelIQTestRun{
		{ChannelID: channel.Id, TestModel: "requested", Engine: "pelican", CompletedAt: 10, Status: model.IQTestSucceeded, HTML: "<html>first</html>"},
		{ChannelID: channel.Id, TestModel: "requested", Engine: "fingerprint", CompletedAt: 11, Status: model.IQTestSucceeded, Analysis: model.JSONValue(`{"prediction":"highest","probability":0.94}`)},
		{ChannelID: channel.Id, TestModel: "requested", Engine: "pelican", CompletedAt: 11, Status: model.IQTestFailed, ErrorMessage: "service unavailable"},
		{ChannelID: channel.Id, TestModel: "another", Engine: "pelican", CompletedAt: 9, Status: model.IQTestSucceeded, HTML: "<html>another</html>"},
		{ChannelID: channel.Id + 1, TestModel: "requested", Engine: "fingerprint", CompletedAt: 12, Status: model.IQTestInconclusive},
	}
	require.NoError(t, db.Create(&runs).Error)
	api := gin.New()
	api.POST("/channel/:id/iq_tests/latest", GetLatestChannelIQTestRuns)
	api.GET("/channel/:id/iq_tests", GetChannelIQTestHistory)
	path := fmt.Sprintf("/channel/%d/iq_tests", channel.Id)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path+"/latest", strings.NewReader(`{"models":["requested","another","missing"]}`)))
	var latest struct {
		Success bool
		Data    []model.ChannelIQTestRun
	}
	require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &latest))
	require.True(t, latest.Success)
	require.Len(t, latest.Data, 2)
	assert.Equal(t, runs[3].ID, latest.Data[0].ID)
	assert.Equal(t, runs[2].ID, latest.Data[1].ID)
	assert.Equal(t, "service unavailable", latest.Data[1].ErrorMessage)
	for page, want := range [][]int64{{runs[2].ID, runs[1].ID}, {runs[0].ID}} {
		rec = httptest.NewRecorder()
		api.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, fmt.Sprintf("%s?model=requested&p=%d&page_size=2", path, page+1), nil))
		var history struct {
			Success bool
			Data    struct {
				Items []model.ChannelIQTestRun
				Total int
				Page  int
			}
		}
		require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &history))
		require.True(t, history.Success)
		assert.Equal(t, page+1, history.Data.Page)
		assert.Equal(t, 3, history.Data.Total)
		require.Len(t, history.Data.Items, len(want))
		for i, id := range want {
			assert.Equal(t, id, history.Data.Items[i].ID)
		}
	}
}

func TestIQTestRequiresARequestedOrConfiguredModel(t *testing.T) {
	db, channel, userID, _, _ := setupChannelRecoveryControllerDB(t)
	require.NoError(t, db.Model(channel).Update("test_model", nil).Error)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/channel/test?iq_test=true", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(channel.Id)}}
	ctx.Set("id", userID)
	TestChannel(ctx)
	var result struct {
		Success bool
		Message string
	}
	require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &result))
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "测试模型")
	var count int64
	require.NoError(t, db.Model(&model.ChannelIQTestRun{}).Count(&count).Error)
	assert.Zero(t, count)
}
