//go:build !plan9
// +build !plan9

package node

import "syscall"

func convertFileLockError(err error) error {
	if errno, ok := err.(syscall.Errno); ok && datadirInUseErrnos[uint(errno)] {
		return ErrDatadirUsed
	}
	return err
}
