package controller

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamChannelProbeEventsForwardsRefreshSignal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/events", StreamChannelProbeEvents)
	server := httptest.NewServer(engine)
	defer server.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(server.URL + "/events")
	require.NoError(t, err)
	defer response.Body.Close()

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))
	assert.Equal(t, "no", response.Header.Get("X-Accel-Buffering"))

	reader := bufio.NewReader(response.Body)
	line, err := reader.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, ": connected\n", line)
	line, err = reader.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "\n", line)

	service.PublishChannelProbeRefresh()
	line, err = reader.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "event: channel-probe-completed\n", line)
	line, err = reader.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "data: refresh\n", line)
}
