package controller

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestModelLeaderboardSyncDueAtFourBeijingTime(t *testing.T) {
	scheduled := time.Date(2026, 11, 22, 20, 0, 0, 0, time.UTC)
	handler := modelLeaderboardSyncHandler{}
	previous := &model.SystemTask{CreatedAt: scheduled.Add(-time.Minute).Unix()}
	today := &model.SystemTask{CreatedAt: scheduled.Unix()}

	assert.False(t, handler.Due(scheduled.Add(-time.Second), nil))
	assert.True(t, handler.Due(scheduled, nil))
	assert.True(t, handler.Due(scheduled.Add(time.Minute), previous))
	assert.False(t, handler.Due(scheduled.Add(time.Minute), today))
	assert.True(t, handler.Due(scheduled.Add(24*time.Hour), today))
}
