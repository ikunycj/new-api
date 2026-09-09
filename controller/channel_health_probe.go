package controller

import (
	"context"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

// RegisterChannelHealthProbeExecutor wires the health probe scheduler to the
// existing channel test implementation.
//
// The scheduler lives in the service package and cannot import controller, so
// the executor is injected here instead. Reusing testChannel rather than
// writing a second request builder matters because it already handles
// provider-specific endpoints, per-channel test models, unsupported channel
// types and streaming quirks; a bespoke probe would drift from real traffic and
// report health that does not match what users experience.
func RegisterChannelHealthProbeExecutor() {
	service.SetChannelProbeExecutor(func(ctx context.Context, channel *model.Channel) (service.ChannelProbeResult, error) {
		if channel == nil {
			return service.ChannelProbeResult{}, nil
		}
		// Probes run on a background schedule with no request context, so the
		// test user is resolved from the root account the same way the existing
		// automatic channel test does.
		testUserID, err := resolveChannelTestUserID(nil)
		if err != nil {
			return service.ChannelProbeResult{}, err
		}

		// Every channel in a family is probed with the same cheap model so their
		// scores stay comparable, falling back to the channel's own model list
		// when it does not serve that model. Probing a channel with a model it
		// does not carry would record a failure that says nothing about health.
		configuredTestModel := ""
		if channel.TestModel != nil {
			configuredTestModel = *channel.TestModel
		}
		probeModel, _ := common.SelectProbeModel(configuredTestModel, channel.GetModels())

		startedAt := time.Now()
		result := testChannel(ctx, channel, testUserID, probeModel, "", shouldUseStreamForAutomaticChannelTest(channel))
		latency := time.Since(startedAt)

		// A local error means the probe itself could not run (unsupported
		// channel type, missing test user). That is not evidence about upstream
		// availability, so it is reported as an error and the caller decides.
		if result.localErr != nil {
			return service.ChannelProbeResult{Latency: latency, ModelName: probeModel}, result.localErr
		}
		if result.newAPIError != nil {
			return service.ChannelProbeResult{Latency: latency, ModelName: probeModel}, result.newAPIError
		}
		return service.ChannelProbeResult{Success: true, Latency: latency, ModelName: probeModel}, nil
	})
}
