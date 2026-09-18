package controller

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func probeTestChannel(id, periodMinutes int) *model.Channel {
	return &model.Channel{
		Id: id, Type: constant.ChannelTypeOpenAI,
		Status: common.ChannelStatusAutoDisabled, AutoProbeEnabled: common.GetPointer(true),
		TestModel: common.GetPointer("gpt-4o"), ProbePeriodMinutes: periodMinutes,
	}
}

func TestChannelProbeSchedulerRunsPeriodGroupsOnMinuteCycles(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ticks := make(chan time.Time)
		var mu sync.Mutex
		starts := map[int]int{}
		scheduler := channelProbeScheduler{
			enabled: func() bool { return true },
			loadCandidates: func(context.Context) ([]*model.Channel, error) {
				return []*model.Channel{
					probeTestChannel(1, 1),
					probeTestChannel(2, 2),
					probeTestChannel(3, 2),
					probeTestChannel(4, 3),
				}, nil
			},
			probe: func(_ context.Context, id int) channelProbeResult {
				mu.Lock()
				starts[id]++
				mu.Unlock()
				return channelProbeResult{summary: channelProbeSummary{Checked: 1, Succeeded: 1}}
			},
		}
		done := make(chan channelProbeResult, 1)
		go func() {
			summary, err := scheduler.run(ctx, ticks)
			done <- channelProbeResult{summary: summary, err: err}
		}()
		synctest.Wait()
		assert.Empty(t, starts, "the scheduler waits for the first 60-second cycle")

		ticks <- time.Now()
		synctest.Wait()
		assert.Equal(t, map[int]int{1: 1}, starts)

		ticks <- time.Now()
		synctest.Wait()
		assert.Equal(t, map[int]int{1: 2, 2: 1, 3: 1}, starts)

		ticks <- time.Now()
		synctest.Wait()
		assert.Equal(t, map[int]int{1: 3, 2: 1, 3: 1, 4: 1}, starts)

		cancel()
		synctest.Wait()
		result := <-done
		require.ErrorIs(t, result.err, context.Canceled)
		assert.Equal(t, channelProbeSummary{Checked: 6, Succeeded: 6}, result.summary)
	})
}

func TestChannelProbeSchedulerLaunchesWholeDueGroupWithoutConcurrencyLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ticks := make(chan time.Time)
		channels := make([]*model.Channel, 0, 32)
		for id := 1; id <= 32; id++ {
			channels = append(channels, probeTestChannel(id, 1))
		}
		started := make(chan int, len(channels))
		scheduler := channelProbeScheduler{
			enabled:        func() bool { return true },
			loadCandidates: func(context.Context) ([]*model.Channel, error) { return channels, nil },
			probe: func(ctx context.Context, id int) channelProbeResult {
				started <- id
				<-ctx.Done()
				return channelProbeResult{}
			},
		}
		done := make(chan error, 1)
		go func() { _, err := scheduler.run(ctx, ticks); done <- err }()
		ticks <- time.Now()
		synctest.Wait()
		assert.Len(t, started, len(channels), "all channels in the due group start together")

		// A later due cycle must not overlap an already-running probe for the same channel.
		ticks <- time.Now()
		synctest.Wait()
		assert.Len(t, started, len(channels))

		cancel()
		synctest.Wait()
		require.ErrorIs(t, <-done, context.Canceled)
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
				var closeOnce sync.Once
				scheduler := channelProbeScheduler{
					enabled: func() bool { return enabled },
					loadCandidates: func(context.Context) ([]*model.Channel, error) {
						return []*model.Channel{probeTestChannel(1, 1)}, nil
					},
					probe: func(ctx context.Context, _ int) channelProbeResult {
						<-ctx.Done()
						closeOnce.Do(func() { close(exited) })
						return channelProbeResult{}
					},
					report: func(channelProbeSummary, int) error { return reportErr },
				}
				done := make(chan error, 1)
				go func() { _, err := scheduler.run(context.Background(), ticks); done <- err }()
				ticks <- time.Now()
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

func TestChannelProbeSchedulerReturnsCandidateLoadError(t *testing.T) {
	wantErr := errors.New("load failed")
	ticks := make(chan time.Time, 1)
	ticks <- time.Now()
	scheduler := channelProbeScheduler{
		enabled:        func() bool { return true },
		loadCandidates: func(context.Context) ([]*model.Channel, error) { return nil, wantErr },
		probe:          func(context.Context, int) channelProbeResult { return channelProbeResult{} },
	}
	_, err := scheduler.run(context.Background(), ticks)
	require.ErrorIs(t, err, wantErr)
}

func TestChannelProbeRandomDelayIsOptionalAndBoundedToOneMinute(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		called := 0
		executor := channelProbeExecutor{randomDelay: func() time.Duration {
			called++
			return 2 * time.Minute
		}}
		assert.True(t, executor.waitRandomDelay(context.Background(), false))
		assert.Zero(t, called)

		done := make(chan bool, 1)
		go func() { done <- executor.waitRandomDelay(context.Background(), true) }()
		synctest.Wait()
		assert.Equal(t, 1, called)
		time.Sleep(channelProbeRandomDelayWindow - time.Second)
		synctest.Wait()
		assert.Empty(t, done)
		time.Sleep(time.Second)
		synctest.Wait()
		assert.True(t, <-done)
	})
}
