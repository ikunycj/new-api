package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPlaygroundRoutesShareGroupOverrideHandling(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "chat completions", path: "/pg/chat/completions", want: true},
		{name: "image generations", path: "/pg/images/generations", want: true},
		{name: "public image generations", path: "/v1/images/generations", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", test.path, nil)
			assert.Equal(t, test.want, isPlaygroundRequest(request.URL.Path))
		})
	}
}

func TestDistributeBypassesChannelSelectionForMockRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Distribute())
	nextCalled := false
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"model-with-no-real-channel"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(constant.MockLoadTestHeader, "true")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusNoContent, response.Code)
}
