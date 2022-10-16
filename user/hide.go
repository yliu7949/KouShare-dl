//go:build !windows

package user

import (
	"os"
	"path/filepath"
	"strings"
)

func hideFile(filename string) error {
	if !strings.HasPrefix(filepath.Base(filename), ".") {
		err := os.Rename(filename, filepath.Dir(filename)+string(os.PathSeparator)+"."+filepath.Base(filename))
		if err != nil {
			return err
		}
	}
	return nil
}
