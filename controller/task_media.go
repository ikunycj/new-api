package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

// TaskMedia streams a media item from a task through the gateway. The public
// response contains only this local URL, so the upstream media origin stays
// private while browser and API clients retain access to the result.
func TaskMedia(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}
	task, exists, err := model.GetByTaskId(c.GetInt("id"), taskID)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("failed to query task media task_id=%s: %v", taskID, err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query task"})
		return
	}
	if !exists || task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task_not_found"})
		return
	}
	if task.Platform != constant.TaskPlatformSuno {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_media_not_supported"})
		return
	}

	index, err := strconv.Atoi(c.DefaultQuery("index", "0"))
	if err != nil || index < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_media_index"})
		return
	}
	kind := strings.ToLower(strings.TrimSpace(c.Query("kind")))
	if kind != "audio" && kind != "video" && kind != "image" && kind != "image_large" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_media_kind"})
		return
	}

	var songs []dto.SunoSong
	if err := common.Unmarshal(task.Data, &songs); err != nil || index >= len(songs) {
		c.JSON(http.StatusNotFound, gin.H{"error": "task_media_not_found"})
		return
	}
	mediaURL := ""
	switch kind {
	case "audio":
		mediaURL = songs[index].AudioURL
	case "video":
		mediaURL = songs[index].VideoURL
	case "image":
		mediaURL = songs[index].ImageURL
	case "image_large":
		mediaURL = songs[index].ImageLargeURL
	}
	streamTaskMedia(c, task, mediaURL, kind)
}

func streamTaskMedia(c *gin.Context, task *model.Task, mediaURL, mediaKind string) {
	mediaURL = strings.TrimSpace(mediaURL)
	if mediaURL == "" || strings.HasPrefix(mediaURL, "data:") {
		c.JSON(http.StatusNotFound, gin.H{"error": "task_media_not_found"})
		return
	}

	channel, err := model.CacheGetChannel(task.ChannelId)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("failed to get task media channel task_id=%s: %v", task.TaskID, err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to retrieve media"})
		return
	}
	client := service.GetSSRFProtectedHTTPClient()
	proxy := channel.GetSetting().Proxy
	if proxy != "" {
		client, err = service.GetHttpClientWithProxy(proxy)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create media client"})
			return
		}
	}

	var validateErr error
	if proxy == "" {
		validateErr = service.ValidateSSRFProtectedFetchURL(mediaURL)
	} else {
		fetchSetting := system_setting.GetFetchSetting()
		validateErr = common.ValidateURLWithFetchSetting(
			mediaURL,
			fetchSetting.EnableSSRFProtection,
			fetchSetting.AllowPrivateIp,
			fetchSetting.DomainFilterMode,
			fetchSetting.IpFilterMode,
			fetchSetting.DomainList,
			fetchSetting.IpList,
			fetchSetting.AllowedPorts,
			fetchSetting.ApplyIPFilterForDomain,
		)
	}
	if validateErr != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("task media request blocked task_id=%s kind=%s: %v", task.TaskID, mediaKind, validateErr))
		c.JSON(http.StatusForbidden, gin.H{"error": "request_blocked"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create media request"})
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("failed to fetch task media task_id=%s kind=%s: %v", task.TaskID, mediaKind, err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch media"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch media"})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		switch mediaKind {
		case "audio":
			contentType = "audio/mpeg"
		case "video":
			contentType = "video/mp4"
		default:
			contentType = "image/jpeg"
		}
	}
	c.Header("Content-Type", contentType)
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		c.Header("Content-Length", contentLength)
	}
	if _, err = io.Copy(c.Writer, resp.Body); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("failed to stream task media task_id=%s kind=%s: %v", task.TaskID, mediaKind, err))
	}
}
