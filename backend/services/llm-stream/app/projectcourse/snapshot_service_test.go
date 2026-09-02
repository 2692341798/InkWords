package projectcourse

import (
	"context"
	"testing"
	"time"
)

type fakeCommitResolver struct {
	sha    string
	branch string
}

func (f fakeCommitResolver) ResolveCommit(context.Context, string, string) (string, string, error) {
	return f.sha, f.branch, nil
}

func TestSnapshotServiceCapturesAndValidatesResolvedSHA(t *testing.T) {
	service := SnapshotService{
		Resolver: fakeCommitResolver{sha: "ABCDEF0123456789ABCDEF0123456789ABCDEF01", branch: "main"},
		Clock:    func() time.Time { return time.Unix(123, 0) },
	}
	snapshot, err := service.Capture(context.Background(), "https://github.com/example/project", "main")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ResolvedCommitSHA != "abcdef0123456789abcdef0123456789abcdef01" {
		t.Fatalf("SHA should be normalized: %q", snapshot.ResolvedCommitSHA)
	}
	if snapshot.DefaultBranch != "main" || snapshot.CapturedAt.Unix() != 123 {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
}

func TestSnapshotServiceRejectsHeadFromResolver(t *testing.T) {
	service := SnapshotService{Resolver: fakeCommitResolver{sha: "HEAD"}}
	if _, err := service.Capture(context.Background(), "https://github.com/example/project", "main"); err == nil {
		t.Fatal("symbolic HEAD must never become a course snapshot")
	}
}
