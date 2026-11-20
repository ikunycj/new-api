package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIQSnapshotBackfillPreservesKnownResultsOnce(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &ChannelIQTestRun{})
	snapshot := `{"iq_probe_engine":"fingerprint","iq_probe_test_model":"original","iq_probe_prediction":"detected","iq_probe_confidence":0,"iq_probe_at":100}`
	require.NoError(t, DB.Create(&[]Channel{{Id: 1, OtherInfo: snapshot}, {Id: 2, OtherInfo: snapshot}, {Id: 3, OtherInfo: `{"iq_probe_test_model":"unknown","iq_probe_at":50}`}}).Error)
	require.NoError(t, DB.Create(&ChannelIQTestRun{ChannelID: 2, TestModel: "original", Engine: "fingerprint", Status: IQTestSucceeded, Source: "manual", CompletedAt: 100}).Error)
	require.NoError(t, backfillChannelIQSnapshots())
	require.NoError(t, backfillChannelIQSnapshots())
	var runs []ChannelIQTestRun
	require.NoError(t, DB.Order("channel_id").Find(&runs).Error)
	require.Len(t, runs, 2)
	assert.Equal(t, 1, runs[0].ChannelID)
	assert.Equal(t, "original", runs[0].TestModel)
	assert.Equal(t, "snapshot", runs[0].Source)
	assert.EqualValues(t, 100, runs[0].CompletedAt)
	assert.Nil(t, runs[0].DurationMs)
	assert.JSONEq(t, `{"prediction":"detected","probability":0,"decision":"scored"}`, string(runs[0].Analysis))
	assert.Equal(t, "manual", runs[1].Source)
}
