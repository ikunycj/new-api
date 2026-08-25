package service

import (
	"context"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/bytedance/gopkg/util/gopool"
)

const channelProbeRefreshRedisChannel = "gateway:channel-probe-refresh:v1"

var (
	channelProbeRefreshSource = common.GetRandomString(16)

	channelProbeRefreshRedisOnce sync.Once

	channelProbeRefreshMu          sync.RWMutex
	channelProbeRefreshNextID      uint64
	channelProbeRefreshSubscribers = map[uint64]chan struct{}{}
)

// StartChannelProbeRefreshSubscriber relays probe refresh signals from other
// application instances into this process. Redis Pub/Sub is optional; without
// Redis, connected clients on the probing instance still receive the signal.
func StartChannelProbeRefreshSubscriber() {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}

	channelProbeRefreshRedisOnce.Do(func() {
		pubsub := common.RDB.Subscribe(context.Background(), channelProbeRefreshRedisChannel)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := pubsub.Receive(ctx)
		cancel()
		if err != nil {
			_ = pubsub.Close()
			common.SysError("subscribe to channel probe refresh events failed: " + err.Error())
			return
		}

		gopool.Go(func() {
			defer pubsub.Close()
			for message := range pubsub.Channel() {
				if message.Payload != channelProbeRefreshSource {
					broadcastChannelProbeRefresh()
				}
			}
		})
	})
}

// SubscribeChannelProbeRefresh returns a coalescing notification stream. A
// slow client needs only one signal because it reloads the complete list.
func SubscribeChannelProbeRefresh() (<-chan struct{}, func()) {
	channelProbeRefreshMu.Lock()
	channelProbeRefreshNextID++
	id := channelProbeRefreshNextID
	events := make(chan struct{}, 1)
	channelProbeRefreshSubscribers[id] = events
	channelProbeRefreshMu.Unlock()

	StartChannelProbeRefreshSubscriber()

	var unsubscribeOnce sync.Once
	unsubscribe := func() {
		unsubscribeOnce.Do(func() {
			channelProbeRefreshMu.Lock()
			delete(channelProbeRefreshSubscribers, id)
			channelProbeRefreshMu.Unlock()
		})
	}
	return events, unsubscribe
}

// PublishChannelProbeRefresh notifies local clients immediately and relays the
// signal through Redis so clients connected to another instance refresh too.
func PublishChannelProbeRefresh() {
	broadcastChannelProbeRefresh()
	if !common.RedisEnabled || common.RDB == nil {
		return
	}

	client := common.RDB
	gopool.Go(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := client.Publish(ctx, channelProbeRefreshRedisChannel, channelProbeRefreshSource).Err(); err != nil {
			common.SysError("publish channel probe refresh event failed: " + err.Error())
		}
	})
}

func broadcastChannelProbeRefresh() {
	channelProbeRefreshMu.RLock()
	defer channelProbeRefreshMu.RUnlock()
	for _, events := range channelProbeRefreshSubscribers {
		select {
		case events <- struct{}{}:
		default:
		}
	}
}
