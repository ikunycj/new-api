package model

import (
	"fmt"
	"strings"
)

// formatChannelDisplayName keeps channel chart dimensions unique by including
// the persisted channel ID in the label. The ID suffix also keeps historical
// rows matchable after a channel has been deleted.
func formatChannelDisplayName(channelID int, channelName string) string {
	name := strings.TrimSpace(channelName)
	if channelID <= 0 {
		if name != "" {
			return name
		}
		return "未记录渠道"
	}

	fallback := fmt.Sprintf("渠道 #%d", channelID)
	if name == "" || name == fallback || name == fmt.Sprintf("channel-%d", channelID) {
		return fallback
	}

	suffix := fmt.Sprintf(" #%d", channelID)
	if strings.HasSuffix(name, suffix) {
		return name
	}
	return name + suffix
}
