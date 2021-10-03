//go:build !windows
// +build !windows

package choria

func (fw *Framework) openLogfile() error {
	return fw.commonLogOpener()
}
