package relay

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func RelaySunoMedia(c *gin.Context) {
	task, exists, err := model.GetByTaskId(c.GetInt("id"), c.Param("task_id"))
	if err != nil || !exists || task == nil || string(task.Platform) != "suno" {
		c.JSON(http.StatusNotFound, gin.H{"error": common.DefaultPublicChannelErrorMessage()})
		return
	}
	field := map[string]string{"audio": "audio_url", "video": "video_url", "image": "image_url", "image_large": "image_large_url"}[strings.ToLower(c.Param("kind"))]
	if field == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.DefaultPublicChannelErrorMessage()})
		return
	}
	var clips []map[string]json.RawMessage
	if err := common.Unmarshal(task.Data, &clips); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": common.DefaultPublicChannelErrorMessage()})
		return
	}
	var mediaURL string
	for _, clip := range clips {
		if common.JsonRawMessageToString(clip["id"]) != c.Param("clip_id") {
			continue
		}
		mediaURL = common.JsonRawMessageToString(clip[field])
		break
	}
	if mediaURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": common.DefaultPublicChannelErrorMessage()})
		return
	}
	channel, err := model.CacheGetChannel(task.ChannelId)
	if err != nil || channel == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": common.DefaultPublicChannelErrorMessage()})
		return
	}
	client := service.GetSSRFProtectedHTTPClient()
	proxy := channel.GetSetting().Proxy
	if proxy != "" {
		client, err = service.GetHttpClientWithProxy(proxy)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": common.DefaultPublicChannelErrorMessage()})
			return
		}
	}
	var validateErr error
	if proxy == "" {
		validateErr = service.ValidateSSRFProtectedFetchURL(mediaURL)
	} else {
		fetchSetting := system_setting.GetFetchSetting()
		validateErr = common.ValidateURLWithFetchSetting(mediaURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain)
	}
	if validateErr != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": common.DefaultPublicChannelErrorMessage()})
		return
	}
	resp, err := client.Get(mediaURL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": common.DefaultPublicChannelErrorMessage()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(resp.StatusCode, gin.H{"error": common.DefaultPublicChannelErrorMessage()})
		return
	}
	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		c.Header("Content-Type", contentType)
	}
	_, _ = io.Copy(c.Writer, resp.Body)
}
