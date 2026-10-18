//go:build unix

package service

import (
	"os"
	"syscall"
)

// fileLogIdentity returns the (device, inode) pair of a file, which is what
// makes a rotated file distinguishable from an appended one: after a rename the
// path points at a different inode while the old file keeps being written to.
// Offset alone cannot tell those apart.
func fileLogIdentity(info os.FileInfo) (uint64, uint64) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0
	}
	return uint64(stat.Dev), uint64(stat.Ino)
}
