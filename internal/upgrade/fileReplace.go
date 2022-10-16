//go:build !windows

package upgrade

import (
	"fmt"
	"os"
	"os/exec"
)

func fileReplace() {
	_ = os.Remove(ksFilePath + ksFileName)
	_ = os.Rename(ksFilePath+ksFileName+".new", ksFilePath+ksFileName)

	cmd := exec.Command(ksFilePath+ksFileName, "version")
	output, _ := cmd.Output()
	fmt.Println(string(output))
}
