package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestClusterCircuitOpensAndRecoversAfterSuccess(t *testing.T) {
	policy := model.DefaultRuntimeFailoverPolicy(model.FailoverModeBalanced)
	policy.CircuitFailureThreshold = 2
	route := "/test/circuit"
	clusterID := 991
	RecordClusterCircuitSuccess(clusterID, route)

	assert.True(t, ClusterCircuitAllows(clusterID, route, policy))
	RecordClusterCircuitFailure(clusterID, route, policy)
	assert.True(t, ClusterCircuitAllows(clusterID, route, policy))
	RecordClusterCircuitFailure(clusterID, route, policy)
	assert.False(t, ClusterCircuitAllows(clusterID, route, policy))

	RecordClusterCircuitSuccess(clusterID, route)
	assert.True(t, ClusterCircuitAllows(clusterID, route, policy))
}
