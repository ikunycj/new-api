package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFailoverMonitoringRejectsInvalidClusterCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, clusterCode := range []string{"0", "-1", "invalid"} {
		t.Run(clusterCode, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/channel/failover/monitoring?cluster_code="+clusterCode, nil)

			GetFailoverMonitoring(ctx)

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			var response struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.False(t, response.Success)
			assert.Equal(t, "cluster_code must be a positive integer", response.Message)
		})
	}
}

func TestGetFailoverMonitoringReturnsSelectedClusterCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, name := range []string{
		"FAILOVER_PROMETHEUS_URL",
		"FAILOVER_ALERTMANAGER_URL",
		"FAILOVER_GRAFANA_PUBLIC_URL",
	} {
		t.Setenv(name, "")
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/channel/failover/monitoring?cluster_code=7", nil)

	GetFailoverMonitoring(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			ClusterCode int `json:"cluster_code"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, 7, response.Data.ClusterCode)
}

func TestDeleteClusterConfigurationRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "invalid"}}
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/channel/failover/cluster-config/invalid", nil)

	DeleteClusterConfiguration(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Equal(t, "invalid cluster ID", response.Message)
}
