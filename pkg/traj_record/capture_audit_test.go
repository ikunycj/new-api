package traj_record

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Opt-in validation of a deployed capture. Only aggregate counts are logged;
// the fixture contains private traffic and must never be committed.
func TestAuditDeployedCapture(t *testing.T) {
	path := os.Getenv("TRAJ_CAPTURE_FILE")
	if path == "" {
		t.Skip("TRAJ_CAPTURE_FILE not set")
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	bodies, calls := reconstruct(t, data)
	require.NotEmpty(t, calls)
	var referenced, complete, truncated int
	for rid, rec := range calls {
		require.False(t, rec.ReqCompacted)
		require.Equal(t, rec.ReqSha256, sha(bodies[rid]), "request digest")
		require.Equal(t, rec.ReqSize, len(bodies[rid]), "request byte count")
		if rec.ParentRid != "" {
			referenced++
		}
		if rec.RespComplete {
			complete++
		}
		if rec.RespTruncated {
			truncated++
		}
	}
	t.Logf("calls=%d exact_requests=%d referenced=%d complete=%d truncated=%d", len(calls), len(bodies), referenced, complete, truncated)
}
