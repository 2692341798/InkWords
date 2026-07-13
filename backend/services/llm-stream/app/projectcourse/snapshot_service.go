package projectcourse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"inkwords-backend/shared/kernel/projectcourse"
)

// CommitResolver 是 GitHub API 或只读 Git 适配器的窄接口。
// 实现不得执行目标仓库的构建、测试、安装脚本或 hook。
type CommitResolver interface {
	ResolveCommit(ctx context.Context, repositoryURL, requestedRef string) (commitSHA, defaultBranch string, err error)
}

type SnapshotService struct {
	Resolver CommitResolver
	Clock    func() time.Time
}

func (s SnapshotService) Capture(ctx context.Context, repositoryURL, requestedRef string) (projectcourse.SourceSnapshot, error) {
	if s.Resolver == nil {
		return projectcourse.SourceSnapshot{}, fmt.Errorf("commit resolver is required")
	}
	if strings.TrimSpace(repositoryURL) == "" || strings.TrimSpace(requestedRef) == "" {
		return projectcourse.SourceSnapshot{}, fmt.Errorf("repository URL and requested ref are required")
	}
	sha, defaultBranch, err := s.Resolver.ResolveCommit(ctx, repositoryURL, requestedRef)
	if err != nil {
		return projectcourse.SourceSnapshot{}, fmt.Errorf("resolve commit: %w", err)
	}
	clock := s.Clock
	if clock == nil {
		clock = time.Now
	}
	snapshot := projectcourse.SourceSnapshot{
		RepositoryURL:     repositoryURL,
		RequestedRef:      requestedRef,
		ResolvedCommitSHA: strings.ToLower(strings.TrimSpace(sha)),
		CapturedAt:        clock().UTC(),
		DefaultBranch:     defaultBranch,
	}
	if err := snapshot.Validate(); err != nil {
		return projectcourse.SourceSnapshot{}, fmt.Errorf("invalid resolved snapshot: %w", err)
	}
	return snapshot, nil
}
