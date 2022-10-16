//go:build windows

package upgrade

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func fileReplace() {
	_ = os.Rename(ksFilePath+ksFileName, ksOldFile)
	_ = os.Rename(ksFilePath+ksFileName+".new", ksFilePath+ksFileName)

	cmd := exec.Command(ksFilePath+ksFileName, "version")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, _ := cmd.Output()
	fmt.Println(string(output))
}
