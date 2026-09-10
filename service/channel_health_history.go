package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/bytedance/gopkg/util/gopool"
)

// The history writer samples the in-memory health snapshots on a timer and folds
// each one into its time bucket. It exists because the live score lives only in
// process memory: a page reload, a second browser or a restart would otherwise
// have no trend to show.
//
// Sampling, not eventing, is the right shape here. A health score is a gauge
// derived from a decaying average, so its value changes continuously with time
// even when no request arrives -- there is no discrete "score changed" moment to
// hook. Reading the snapshot on a timer also keeps this entirely off the request
// path: nothing in relay or routing waits on a database write.
const (
	// channelHealthHistoryTickSeconds is how often snapshots are folded into the
	// current bucket. It is finer than the default bucket so a bucket receives
	// several observations and its mean is not a single instant.
	channelHealthHistoryTickSeconds = 20
)

var channelHealthHistoryOnce sync.Once

// StartChannelHealthHistoryTask launches the sampler on the master node only.
// Followers would write duplicate observations into the same bucket; the mean
// would still be correct, but observation_count would be inflated and the
// confident ratio distorted.
func StartChannelHealthHistoryTask() {
	channelHealthHistoryOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), "channel health history task started")
			ticker := time.NewTicker(channelHealthHistoryTickSeconds * time.Second)
			defer ticker.Stop()
			lastCleanup := time.Time{}
			for range ticker.C {
				RunChannelHealthHistoryOnce()
				// Retention is enforced hourly rather than every tick: it is a
				// range delete over an indexed column, cheap but pointless to
				// repeat every 20 seconds.
				if time.Since(lastCleanup) >= time.Hour {
					cleanupChannelHealthHistory()
					lastCleanup = time.Now()
				}
			}
		})
	})
}

// RunChannelHealthHistoryOnce folds the current snapshots into their bucket. It
// is exported so tests can drive it deterministically instead of waiting on the
// ticker.
func RunChannelHealthHistoryOnce() {
	if !common.IsChannelHealthHistoryEnabled() {
		return
	}
	snapshots := ListChannelHealthSnapshots()
	if len(snapshots) == 0 {
		return
	}

	bucketSeconds := common.ChannelHealthHistoryBucketSeconds()
	bucketTs := model.ChannelHealthHistoryBucket(time.Now().Unix(), bucketSeconds)

	for _, snapshot := range snapshots {
		confidentCount := int64(0)
		if snapshot.Confident {
			confidentCount = 1
		}
		entry := &model.ChannelHealthHistory{
			ChannelId:        snapshot.ChannelID,
			Route:            snapshot.Route,
			Family:           snapshot.Family,
			BucketTs:         bucketTs,
			Score:            snapshot.Score,
			Availability:     snapshot.Availability,
			LatencyScore:     snapshot.LatencyScore,
			LatencyMs:        snapshot.LastLatencyMs,
			Samples:          snapshot.Samples,
			ConfidentCount:   confidentCount,
			ObservationCount: 1,
		}
		if err := model.UpsertChannelHealthHistory(entry); err != nil {
			// One failed row must not abort the sweep: the next tick will fold a
			// fresh observation into the same bucket.
			common.SysError(fmt.Sprintf(
				"channel health history: upsert channel=%d route=%s family=%s bucket=%d failed: %v",
				snapshot.ChannelID, snapshot.Route, snapshot.Family, bucketTs, err))
		}
	}
}

func cleanupChannelHealthHistory() {
	retentionDays := common.ChannelHealthHistoryRetentionDays()
	if retentionDays <= 0 {
		return
	}
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).Unix()
	if err := model.DeleteChannelHealthHistoryBefore(cutoff); err != nil {
		common.SysError("channel health history: cleanup failed: " + err.Error())
	}
}
