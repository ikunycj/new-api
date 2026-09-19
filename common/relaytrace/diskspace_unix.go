//go:build !windows

package relaytrace

import "golang.org/x/sys/unix"

// freeDiskBytes reports space available to an unprivileged writer.
//
// Uses Bavail, not Bfree: filesystems reserve a slice of blocks for root, and
// Bfree counts those. Trusting Bfree would let the guard believe there is room
// while ordinary writes already fail with ENOSPC.
func freeDiskBytes(dir string) (uint64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return uint64(st.Bavail) * uint64(st.Bsize), nil
}
