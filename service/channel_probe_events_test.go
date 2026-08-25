package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelProbeRefreshSubscriptionCoalescesAndUnsubscribes(t *testing.T) {
	events, unsubscribe := SubscribeChannelProbeRefresh()
	require.NotNil(t, events)

	PublishChannelProbeRefresh()
	PublishChannelProbeRefresh()

	select {
	case <-events:
	default:
		require.Fail(t, "subscriber did not receive channel probe refresh")
	}
	select {
	case <-events:
		assert.Fail(t, "duplicate channel probe refresh was not coalesced")
	default:
	}

	unsubscribe()
	PublishChannelProbeRefresh()
	select {
	case <-events:
		assert.Fail(t, "unsubscribed listener received channel probe refresh")
	default:
	}
}
