package model

import "time"

// ChannelRuntimeMetrics contains only dynamic values derived from logs or
// process state. Channel configuration and credentials are intentionally not
// part of this snapshot.
type ChannelRuntimeMetrics struct {
	LastProbeAt                 int64        `json:"last_probe_at"`
	LastTestTTFTMs              float64      `json:"last_test_ttft_ms"`
	PreviousDayProbeSuccessRate float64      `json:"previous_day_probe_success_rate"`
	PreviousDayProbeSampleCount int          `json:"previous_day_probe_sample_count"`
	PreviousDayAverageTTFTMs    float64      `json:"previous_day_average_ttft_ms"`
	Usage                       ChannelUsage `json:"usage"`
}

// GetChannelRuntimeMetrics loads one runtime snapshot for the page request.
func GetChannelRuntimeMetrics(channelIDs []int, now time.Time) (map[int]ChannelRuntimeMetrics, error) {
	return loadChannelRuntimeMetrics(channelIDs, now)
}

func loadChannelRuntimeMetrics(channelIDs []int, now time.Time) (map[int]ChannelRuntimeMetrics, error) {
	probeTimes, err := GetLastChannelProbeTimes(channelIDs)
	if err != nil {
		return nil, err
	}
	ttfts, err := GetLatestChannelTestTTFTs(channelIDs)
	if err != nil {
		return nil, err
	}
	rates, samples, err := GetPreviousDayChannelProbeStats(channelIDs, now)
	if err != nil {
		return nil, err
	}
	averages, err := GetPreviousDayChannelAverageTTFTs(channelIDs, now)
	if err != nil {
		return nil, err
	}
	usage, err := GetChannelUsageAt(channelIDs, now)
	if err != nil {
		return nil, err
	}

	result := make(map[int]ChannelRuntimeMetrics, len(channelIDs))
	for _, id := range channelIDs {
		result[id] = ChannelRuntimeMetrics{
			LastProbeAt:                 probeTimes[id],
			LastTestTTFTMs:              ttfts[id],
			PreviousDayProbeSuccessRate: rates[id],
			PreviousDayProbeSampleCount: samples[id],
			PreviousDayAverageTTFTMs:    averages[id],
			Usage:                       usage[id],
		}
	}
	return result, nil
}
