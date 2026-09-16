package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
)

func formatNotifyType(channelId int, status int) string {
	return fmt.Sprintf("%s_%d_%d", dto.NotifyTypeChannelUpdate, channelId, status)
}

// disable & notify
func DisableChannel(channelError types.ChannelError, reason string) {
	common.SysLog(fmt.Sprintf("通道「%s」（#%d）发生错误，准备禁用，原因：%s", channelError.ChannelName, channelError.ChannelId, common.LocalLogPreview(reason)))

	// 检查是否启用自动禁用功能
	if !channelError.AutoBan {
		common.SysLog(fmt.Sprintf("通道「%s」（#%d）未启用自动禁用功能，跳过禁用操作", channelError.ChannelName, channelError.ChannelId))
		return
	}

	success := model.UpdateChannelStatus(channelError.ChannelId, channelError.UsingKey, common.ChannelStatusAutoDisabled, reason)
	if success {
		subject := fmt.Sprintf("通道「%s」（#%d）已被禁用", channelError.ChannelName, channelError.ChannelId)
		content := fmt.Sprintf("通道「%s」（#%d）已被禁用，原因：%s", channelError.ChannelName, channelError.ChannelId, reason)
		NotifyRootUser(formatNotifyType(channelError.ChannelId, common.ChannelStatusAutoDisabled), subject, content)
	}
}

func EnableChannel(channelId int, usingKey string, channelName string) {
	success := model.UpdateChannelStatus(channelId, usingKey, common.ChannelStatusEnabled, "")
	if success {
		subject := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
		content := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
		NotifyRootUser(formatNotifyType(channelId, common.ChannelStatusEnabled), subject, content)
	}
}

// CompleteChannelProbe notifies only after the lease-guarded result and channel
// status have committed together.
func CompleteChannelProbe(ctx context.Context, channelID int, result model.ChannelProbeHistory, leaseUntil int64, usingKey string) (int, bool, error) {
	channel, changed, err := model.CompleteChannelProbe(ctx, channelID, result, leaseUntil, usingKey)
	if err != nil {
		return 0, false, err
	}
	if changed {
		notifyChannelProbeStatus(channel, result.ErrorMessage)
	}
	return channel.Status, changed, nil
}

// RecoverChannelAfterTest 仅在安全恢复事务提交后通知，不使用请求开始时的渠道快照。
func RecoverChannelAfterTest(ctx context.Context, channelID int, usingKey string) error {
	channel, changed, err := model.RecoverChannelAfterTest(ctx, channelID, usingKey)
	if err != nil {
		return err
	}
	if changed {
		notifyChannelProbeStatus(channel, "")
	}
	return nil
}

func notifyChannelProbeStatus(channel *model.Channel, reason string) {
	subject := fmt.Sprintf("通道「%s」（#%d）已被启用", channel.Name, channel.Id)
	content := subject
	if channel.Status == common.ChannelStatusAutoDisabled {
		subject = fmt.Sprintf("通道「%s」（#%d）已被禁用", channel.Name, channel.Id)
		content = subject + "，原因：" + reason
	}
	// 通知投递不占用探测槽位；只有实际状态变化才会通知。
	go NotifyRootUser(formatNotifyType(channel.Id, channel.Status), subject, content)
}

func ShouldDisableChannel(err *types.NewAPIError) bool {
	if !common.AutomaticDisableChannelEnabled {
		return false
	}
	if err == nil {
		return false
	}
	if types.IsChannelError(err) {
		return true
	}
	if types.IsSkipRetryError(err) {
		return false
	}
	if operation_setting.ShouldDisableByStatusCode(err.StatusCode) {
		return true
	}

	lowerMessage := strings.ToLower(err.Error())
	search, _ := AcSearch(lowerMessage, operation_setting.AutomaticDisableKeywords, true)
	return search
}
