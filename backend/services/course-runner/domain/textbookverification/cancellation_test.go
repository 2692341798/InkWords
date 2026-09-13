package textbookverification

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
	"inkwords-backend/shared/platform/teachingartifact"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type cancellationTasks struct {
	recordingTextbookTaskStore
	entered  atomic.Bool
	failRead bool
	executor *cancellableExecutor
	released bool
}

func (s *cancellationTasks) ReleaseVerificationWorker(ctx context.Context, _, token uuid.UUID) error {
	if ctx.Err() != nil || token == uuid.Nil || !s.executor.stopped.Load() {
		return errors.New("release preceded executor exit or used cancelled context")
	}
	s.released = true
	return nil
}

func (s *cancellationTasks) IsCancelled(context.Context, uuid.UUID) (bool, error) {
	if s.entered.Load() && s.failRead {
		return false, errors.New("task store unavailable")
	}
	return s.entered.Load(), nil
}

type cancellableExecutor struct {
	tasks   *cancellationTasks
	fast    bool
	stopped atomic.Bool
}

func (e *cancellableExecutor) Execute(ctx context.Context, _, _ string, _ time.Duration, _ []string) (int, string, error) {
	e.tasks.entered.Store(true)
	if e.fast {
		e.stopped.Store(true)
		return 0, "test completed just as cancel arrived", nil
	}
	<-ctx.Done()
	e.stopped.Store(true)
	return -1, "", ctx.Err()
}

func TestConsumerStopsRunningOrJustFinishedWorkBeforeRecordingEvidence(t *testing.T) {
	for _, test := range []struct {
		name           string
		fast, failRead bool
	}{{"running cancellation", false, false}, {"completion race", true, false}, {"observation failure", false, true}} {
		t.Run(test.name, func(t *testing.T) {
			store := teachingartifact.NewStore(t.TempDir())
			artifact, err := store.Stage(t.Context(), []teachingartifact.File{{Path: "main.go", Content: []byte("package main\n")}})
			require.NoError(t, err)
			root, err := store.Resolve(artifact.Token)
			require.NoError(t, err)
			manifest := ArtifactManifest{Format: ArtifactManifestFormat, ArtifactID: uuid.NewString(), RevisionID: uuid.NewString(), ArtifactHash: artifact.ArtifactHash, BookContractHash: "sha256:" + strings.Repeat("a", 64), StyleSheetHash: "sha256:" + strings.Repeat("b", 64), Language: "go", ToolchainVersion: "go1.26.8", Commands: []CommandTemplate{{Kind: "go_test"}}}
			payload, _ := json.Marshal(VerificationPayload{ArtifactID: manifest.ArtifactID, RevisionID: manifest.RevisionID, ArtifactHash: manifest.ArtifactHash, ManifestHash: "sha256:" + strings.Repeat("c", 64)})
			tasks := &cancellationTasks{failRead: test.failRead}
			executor := &cancellableExecutor{tasks: tasks, fast: test.fast}
			tasks.executor = executor
			evidence := &recordingEvidencePersister{}
			consumer := NewConsumer(tasks, fakeTextbookResolver{request: RunRequest{Manifest: manifest, RootDir: root}}, Runner{Executor: executor, RunnerImageDigest: "sha256:" + strings.Repeat("d", 64)}, evidence)
			workspace := uuid.New()
			ctx, finish := context.WithTimeout(t.Context(), 3*time.Second)
			defer finish()
			err = consumer.HandleVerificationRequested(ctx, sharedrabbitmq.TextbookVerificationRequestedMessage{TaskID: uuid.New(), Kind: TaskSubtype, WorkspaceID: &workspace, Payload: payload})
			if test.failRead {
				require.ErrorContains(t, err, "task store unavailable")
			} else {
				require.NoError(t, err)
			}
			require.False(t, tasks.succeeded)
			require.Empty(t, evidence.report)
			require.Empty(t, tasks.failed)
			require.True(t, tasks.released)
			if !test.fast {
				require.True(t, executor.stopped.Load())
			}
		})
	}
}
