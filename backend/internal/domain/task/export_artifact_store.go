package task

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// ExportArtifactStore 管理共享导出目录中的受控下载文件。
type ExportArtifactStore struct {
	rootDir     string
	ttl         time.Duration
	nowProvider func() time.Time
}

// NewExportArtifactStore 通过依赖注入组装导出产物仓储。
func NewExportArtifactStore(rootDir string, ttl time.Duration, nowProvider func() time.Time) *ExportArtifactStore {
	if nowProvider == nil {
		nowProvider = time.Now
	}
	return &ExportArtifactStore{
		rootDir:     rootDir,
		ttl:         ttl,
		nowProvider: nowProvider,
	}
}

// Save 把临时 PDF 转存到共享目录，并返回供任务结果复用的下载元数据。
func (s *ExportArtifactStore) Save(taskID uuid.UUID, sourcePath string, filename string) (ExportTaskResult, error) {
	token := fmt.Sprintf("exp_pdf_%s", taskID.String())
	targetPath := s.PathForToken(token)
	if err := os.MkdirAll(s.rootDir, 0o755); err != nil {
		return ExportTaskResult{}, err
	}

	// Why: Docker volume 和容器临时目录可能不在同一挂载点，rename 失败时必须回退到 copy。
	if err := moveFile(sourcePath, targetPath); err != nil {
		return ExportTaskResult{}, err
	}

	expiresAt := s.nowProvider().Add(s.ttl).UTC()
	return ExportTaskResult{
		FileToken:   token,
		Filename:    filename,
		ContentType: "application/pdf",
		ExpiresAt:   expiresAt,
	}, nil
}

// PathForToken 根据文件令牌计算共享目录中的绝对路径。
func (s *ExportArtifactStore) PathForToken(token string) string {
	return filepath.Join(s.rootDir, token+".pdf")
}

func moveFile(sourcePath string, targetPath string) error {
	if err := os.Rename(sourcePath, targetPath); err == nil {
		return nil
	}

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		_ = targetFile.Close()
		_ = os.Remove(targetPath)
		return err
	}
	if err := targetFile.Close(); err != nil {
		_ = os.Remove(targetPath)
		return err
	}
	return os.Remove(sourcePath)
}
