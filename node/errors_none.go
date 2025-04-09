//go:build plan9
// +build plan9

package node

func convertFileLockError(err error) error {
	return err
}
