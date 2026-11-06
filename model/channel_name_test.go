package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatChannelDisplayNameUsesStableID(t *testing.T) {
	assert.Equal(t, "ChatGPT Plus #12", formatChannelDisplayName(12, "ChatGPT Plus"))
	assert.Equal(t, "ChatGPT Plus #12", formatChannelDisplayName(12, "ChatGPT Plus #12"))
	assert.Equal(t, "渠道 #12", formatChannelDisplayName(12, "channel-12"))
	assert.Equal(t, "渠道 #12", formatChannelDisplayName(12, ""))
	assert.Equal(t, "未记录渠道", formatChannelDisplayName(0, ""))
}
