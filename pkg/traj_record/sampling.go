package traj_record

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
)

// sampleRequest uses a stable session bucket, independent of request order,
// history prefixes, file rotation, process lifetime, and gateway instance.
// Keep the rate identical across instances and throughout a session.
func (r *recorder) sampleRequest(cl *client, userID int, headers http.Header, body []byte) bool {
	if r.cfg.SampleRate == nil || *r.cfg.SampleRate >= 1 {
		return true
	}
	if *r.cfg.SampleRate <= 0 {
		return false
	}
	var top map[string]json.RawMessage
	_ = common.Unmarshal(body, &top)
	rec := callRecord{Headers: map[string]string{
		"Session-Id": headers.Get("Session-Id"),
		"Session_id": headers.Get("Session_id"),
	}}
	cl.requestIds(&rec, top)
	// A thread id alone cannot establish the root session of a child. Do not
	// independently sample unidentified calls and produce partial families.
	// Keep unknown sessions in full; zero above remains an absolute off switch.
	if rec.SessionId == "" {
		return true // Unknown sessions are retained; request-level sampling would fragment them.
	}
	key, _ := common.Marshal([]string{"traj-session-v1", cl.name, strconv.Itoa(userID), rec.SessionId})
	sum := sha256.Sum256(key)
	// Use 53 bits so every bucket is exactly representable as a float64.
	bucket := float64(binary.BigEndian.Uint64(sum[:8])>>11) / (1 << 53)
	return bucket < *r.cfg.SampleRate
}
