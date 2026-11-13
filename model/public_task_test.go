package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
)

func TestPublicMidjourneyViewPreservesOriginalAndSuccessfulResult(t *testing.T) {
	original := &Midjourney{ChannelId: 77, Status: "SUCCESS", ImageUrl: "https://cdn.example.com/image.png", Prompt: "my prompt", Properties: `{"channel_id":77,"upstream_model_name":"secret-model"}`}
	before := *original
	view := original.PublicMidjourney()
	assert.Equal(t, before, *original)
	assert.Equal(t, 1, view.ChannelId)
	assert.Equal(t, original.ImageUrl, view.ImageUrl)
	assert.Equal(t, original.Prompt, view.Prompt)
	assert.JSONEq(t, `{"channel_id":1}`, view.Properties)
}

func TestPublicMidjourneyViewHidesFailureDescription(t *testing.T) {
	original := &Midjourney{ChannelId: 77, Status: "FAILURE", FailReason: "supplier failed", Description: "channel 77 failed"}
	view := original.PublicMidjourney()
	assert.Equal(t, common.DefaultPublicChannelErrorMessage(), view.FailReason)
	assert.Equal(t, view.FailReason, view.Description)
	assert.Equal(t, "channel 77 failed", original.Description)
}
