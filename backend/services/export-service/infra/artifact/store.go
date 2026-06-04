package artifact

import (
	"time"

	taskdomain "inkwords-backend/internal/domain/task"
)

// NewStore creates the export artifact store with export-service defaults.
func NewStore(dir string) *taskdomain.ExportArtifactStore {
	return taskdomain.NewExportArtifactStore(dir, 15*time.Minute, time.Now)
}
