package model

import (
	"github.com/QuantumNous/new-api/common"
)

// PublicMidjourney makes a response-only copy. The original is still used by
// polling, accounting and administrator views.
func (m *Midjourney) PublicMidjourney() *Midjourney {
	view := *m
	view.ChannelId = common.PublicChannelID
	view.FailReason = ""
	if m.Status == "FAILURE" || (m.Status != "SUCCESS" && m.FailReason != "") {
		_, view.FailReason = common.PublicChannelError("", m.FailReason)
		view.Description = view.FailReason
	}
	if m.Properties != "" {
		data, err := common.PublicTaskJSON([]byte(m.Properties))
		if err != nil {
			common.SysError("cannot project Midjourney properties: " + err.Error())
			view.Properties = "{}"
		} else {
			view.Properties = string(data)
		}
		var properties map[string]any
		if err := common.UnmarshalJsonStr(view.Properties, &properties); err == nil {
			if _, ok := properties["imageUrl"]; ok {
				properties["imageUrl"] = "/mj/image/" + view.MjId
			}
			if _, ok := properties["videoUrl"]; ok {
				properties["videoUrl"] = "/mj/video/" + view.MjId
			}
			if data, err := common.Marshal(properties); err == nil {
				view.Properties = string(data)
			}
		}
	}
	return &view
}
