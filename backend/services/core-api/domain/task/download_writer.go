package task

import (
	"fmt"
	"io"
)

func copyDownload(dst io.Writer, src io.Reader) error {
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("stream download: %w", err)
	}
	return nil
}
