package controller

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func probeTestChannel(id int) *model.Channel {
	return &model.Channel{
		Id: id, Type: constant.ChannelTypeOpenAI,
		Status:           common.ChannelStatusAutoDisabled,
		AutoProbeEnabled: common.GetPointer(true), TestModel: common.GetPointer("gpt-4o"),
	}
}

func TestChannelProbeSchedulerReprobesDueChannelWhileAnotherIsBlocked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ticks := make(chan time.Time)
		var mu sync.Mutex
		starts := map[int]int{}
		nextProbeAt := int64(0)
		scheduler := channelProbeScheduler{
			enabled: func() bool { return true },
			loadDue: func(_ context.Context, now int64) ([]*model.Channel, error) {
				mu.Lock()
				defer mu.Unlock()
				channels := []*model.Channel{probeTestChannel(1)}
				if now >= nextProbeAt {
					channels = append(channels, probeTestChannel(2))
				}
				return channels, nil
			},
			probe: func(ctx context.Context, id int) channelProbeResult {
				mu.Lock()
				starts[id]++
				if id == 2 {
					nextProbeAt = common.GetTimestamp() + 10
				}
				mu.Unlock()
				if id == 1 {
					<-ctx.Done()
					return channelProbeResult{}
				}
				return channelProbeResult{summary: channelProbeSummary{Checked: 1, Failed: 1}}
			},
		}
		done := make(chan channelProbeResult, 1)
		go func() {
			summary, err := scheduler.run(ctx, ticks)
			done <- channelProbeResult{summary: summary, err: err}
		}()
		synctest.Wait()
		assert.Equal(t, map[int]int{1: 1, 2: 1}, starts)

		// Advance the synthetic clock, not wall time: no early retry at 9s.
		time.Sleep(9 * time.Second)
		ticks <- time.Now()
		synctest.Wait()
		assert.Equal(t, map[int]int{1: 1, 2: 1}, starts)
		time.Sleep(time.Second)
		ticks <- time.Now()
		synctest.Wait()
		assert.Equal(t, map[int]int{1: 1, 2: 2}, starts, "channel 2 retries at 10s while channel 1 remains in flight")
		cancel()
		synctest.Wait()
		result := <-done
		require.ErrorIs(t, result.err, context.Canceled)
		assert.Equal(t, channelProbeSummary{Checked: 2, Failed: 2}, result.summary)
	})
}

func TestChannelProbeSchedulerBoundsWorkersAndDrainsOnCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ticks := make(chan time.Time)
		releaseFirst := make(chan struct{})
		var mu sync.Mutex
		started := map[int]int{}
		finished := map[int]bool{}
		scheduler := channelProbeScheduler{
			enabled: func() bool { return true },
			loadDue: func(context.Context, int64) ([]*model.Channel, error) {
				mu.Lock()
				defer mu.Unlock()
				var channels []*model.Channel
				for id := 1; id <= 9; id++ {
					if !finished[id] {
						channels = append(channels, probeTestChannel(id))
					}
				}
				return channels, nil
			},
			probe: func(ctx context.Context, id int) channelProbeResult {
				mu.Lock()
				started[id]++
				mu.Unlock()
				if id == 1 {
					select {
					case <-releaseFirst:
					case <-ctx.Done():
					}
				} else {
					<-ctx.Done()
				}
				mu.Lock()
				finished[id] = true
				mu.Unlock()
				return channelProbeResult{}
			},
		}
		done := make(chan error, 1)
		go func() { _, err := scheduler.run(ctx, ticks); done <- err }()
		synctest.Wait()
		require.Len(t, started, 8)
		assert.Zero(t, started[9])
		ticks <- time.Now()
		synctest.Wait()
		assert.Len(t, started, 8)
		close(releaseFirst)
		synctest.Wait()
		ticks <- time.Now()
		synctest.Wait()
		assert.Equal(t, 1, started[9], "a released slot accepts queued work without waiting for the other probes")
		cancel()
		synctest.Wait()
		require.ErrorIs(t, <-done, context.Canceled)
		assert.Len(t, finished, 9, "all in-flight probes exit before the dispatcher returns")
		for id, count := range started {
			assert.Equal(t, 1, count, "no overlapping probe for channel %d", id)
		}
	})
}

func TestChannelProbeSchedulerStopsOnLeaseLossOrDisable(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "lease lost", true: "disabled"}[disabled], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ticks := make(chan time.Time)
				enabled := true
				var reportErr error
				exited := make(chan struct{})
				scheduler := channelProbeScheduler{
					enabled: func() bool { return enabled },
					loadDue: func(context.Context, int64) ([]*model.Channel, error) {
						return []*model.Channel{probeTestChannel(1)}, nil
					},
					probe: func(ctx context.Context, _ int) channelProbeResult {
						<-ctx.Done()
						close(exited)
						return channelProbeResult{}
					},
					report: func(channelProbeSummary, int) error { return reportErr },
				}
				done := make(chan error, 1)
				go func() { _, err := scheduler.run(context.Background(), ticks); done <- err }()
				synctest.Wait()
				if disabled {
					enabled = false
				} else {
					reportErr = model.ErrSystemTaskLockLost
				}
				ticks <- time.Now()
				synctest.Wait()
				err := <-done
				if disabled {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, model.ErrSystemTaskLockLost)
				}
				select {
				case <-exited:
				default:
					t.Fatal("dispatcher returned before its probe stopped")
				}
			})
		})
	}
}

func TestChannelProbeSchedulerSkipsBusyChannelsWithoutStarvingRecovery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var channels []*model.Channel
		for id := 1; id <= 9; id++ {
			channel := probeTestChannel(91000 + id)
			channel.MaxConcurrency = common.GetPointer(1)
			channels = append(channels, channel)
			if id <= 8 {
				require.True(t, service.TryAcquireChannelConcurrency(channel.Id, 1))
				t.Cleanup(func() { service.ReleaseChannelConcurrency(channel.Id) })
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		started := make(chan int, 9)
		scheduler := channelProbeScheduler{
			enabled: func() bool { return true },
			loadDue: func(context.Context, int64) ([]*model.Channel, error) { return channels, nil },
			probe: func(ctx context.Context, id int) channelProbeResult {
				started <- id
				<-ctx.Done()
				return channelProbeResult{}
			},
		}
		done := make(chan error, 1)
		go func() { _, err := scheduler.run(ctx, make(chan time.Time)); done <- err }()
		synctest.Wait()
		require.Len(t, started, 1)
		assert.Equal(t, 91009, <-started)
		cancel()
		synctest.Wait()
		require.ErrorIs(t, <-done, context.Canceled)
	})
}
