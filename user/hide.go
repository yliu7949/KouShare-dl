// +build !windows

package user

import (
	"os"
	"path/filepath"
	"strings"
)

func hideFile(filename string) error {
	if !strings.HasPrefix(filepath.Base(filename), ".") {
		err := os.Rename(filename, "."+filename)
		if err != nil {
			return err
		}
	}
	return nil
}
