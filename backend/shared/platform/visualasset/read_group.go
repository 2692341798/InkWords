package visualasset

import (
	"fmt"
	"os"
)

// WithReadGroup allows a trusted publisher to grant an existing export reader
// group access. Default stores stay private; readers still use read-only mounts.
func (store *Store) WithReadGroup(gid int) *Store {
	store.readGroup = &gid
	return store
}

func setVisualReadPermission(path string, directory bool, gid int) error {
	if gid < 1 {
		return fmt.Errorf("invalid visual asset reader group")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || info.IsDir() != directory || (!directory && !info.Mode().IsRegular()) {
		return ErrInvalidToken
	}
	if err := os.Chown(path, -1, gid); err != nil {
		return fmt.Errorf("set visual asset reader group: %w", err)
	}
	mode := os.FileMode(0o640)
	if directory {
		mode = 0o750
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("set visual asset read permission: %w", err)
	}
	return nil
}
