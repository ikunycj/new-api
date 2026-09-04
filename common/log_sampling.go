package common

import (
	"hash/fnv"
	"math"
	"sync/atomic"
)

var accessLogSuccessSampleRateBits atomic.Uint64

func init() {
	accessLogSuccessSampleRateBits.Store(math.Float64bits(1))
}

func SetAccessLogSuccessSampleRate(rate float64) {
	if math.IsNaN(rate) || math.IsInf(rate, 0) {
		rate = 1
	}
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	accessLogSuccessSampleRateBits.Store(math.Float64bits(rate))
}

func GetAccessLogSuccessSampleRate() float64 {
	return math.Float64frombits(accessLogSuccessSampleRateBits.Load())
}

// ShouldLogAccessRequest keeps failures and slow requests while sampling fast successes.
func ShouldLogAccessRequest(statusCode int, latencyMs int64, requestID, path string) bool {
	if statusCode >= 400 || latencyMs >= 1000 {
		return true
	}
	rate := GetAccessLogSuccessSampleRate()
	if rate >= 1 {
		return true
	}
	if rate <= 0 {
		return false
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(requestID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(path))
	return float64(h.Sum32())/float64(math.MaxUint32) < rate
}
