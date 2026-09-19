//go:build windows

package relaytrace

import "golang.org/x/sys/windows"

// freeDiskBytes reports space available to the calling user.
//
// GetDiskFreeSpaceEx's first output is the caller-visible free space, which
// already accounts for per-user quotas; the volume-wide total is deliberately
// ignored for the same reason Bavail is preferred over Bfree on unix.
func freeDiskBytes(dir string) (uint64, error) {
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return 0, err
	}
	var availToCaller, total, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &availToCaller, &total, &totalFree); err != nil {
		return 0, err
	}
	return availToCaller, nil
}
