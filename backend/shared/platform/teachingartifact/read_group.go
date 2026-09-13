package teachingartifact

import (
	"fmt"
	"os"
	"path/filepath"
)

func setReaderPermission(path string, directory bool, gid int) error {
	if gid < 1 {
		return fmt.Errorf("invalid teaching artifact reader group")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
		return ErrUnsafeFile
	}
	if err := os.Chown(path, -1, gid); err != nil {
		return fmt.Errorf("set teaching artifact reader group: %w", err)
	}
	mode := os.FileMode(0o640)
	if directory {
		mode = 0o750
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("set teaching artifact read permission: %w", err)
	}
	return nil
}

func shareReadTree(root string, gid int) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return setReaderPermission(path, info.IsDir(), gid)
	})
}
