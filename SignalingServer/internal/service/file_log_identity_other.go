//go:build !unix

package service

import "os"

// On platforms without a stable inode (Windows) there is no cheap way to tell a
// rotated file from an appended one, so the identity is empty and rotation is
// detected only by the size shrinking. A rename-plus-recreate that immediately
// grows past the old offset is therefore missed on those platforms - which is
// why the tailer is documented for the Linux/macOS deployment, where the
// PostgreSQL and Redis logs are rotated by rename.
func fileLogIdentity(os.FileInfo) (uint64, uint64) {
	return 0, 0
}
