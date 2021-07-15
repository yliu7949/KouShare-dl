// +build windows

package user

import (
	"syscall"
)

func hideFile(filename string) error {
	filenameW, err := syscall.UTF16PtrFromString(filename)
	if err != nil {
		return err
	}
	err = syscall.SetFileAttributes(filenameW, syscall.FILE_ATTRIBUTE_HIDDEN)
	if err != nil {
		return err
	}
	return nil
}
