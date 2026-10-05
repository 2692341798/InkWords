package textbook

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLearnerFilesPreserveBytesAndRejectUnsafeOrOversizedSubmissions(t *testing.T) {
	input := []LearnerCodeFile{{Path: "router_test.go", Content: "package main\r\n"}, {Path: "router.go", Content: "package main\n// 我写的实现\n"}}
	files, err := NormalizeLearnerFiles(input)
	require.NoError(t, err)
	require.Equal(t, "router.go", files[0].Path)
	require.Equal(t, input[0].Content, files[1].Content)
	a, err := LearnerFilesHash(input)
	require.NoError(t, err)
	b, err := LearnerFilesHash(files)
	require.NoError(t, err)
	require.Equal(t, a, b)
	files[0].Content += "\n"
	b, err = LearnerFilesHash(files)
	require.NoError(t, err)
	require.NotEqual(t, a, b)
	for _, path := range []string{"../escape.go", "/tmp/answer.go", "a/../answer.go", "a\\answer.go", ".git/config", "run.sh", "a//b.go", " go.mod", "go.work", "vendor/a.go", "a/.hidden.go"} {
		_, err := NormalizeLearnerFiles([]LearnerCodeFile{{Path: path, Content: "package main"}})
		require.Error(t, err, path)
	}
	_, err = NormalizeLearnerFiles([]LearnerCodeFile{{Path: "a.go", Content: "package main"}, {Path: "a.go", Content: "different"}})
	require.Error(t, err)
	_, err = NormalizeLearnerFiles([]LearnerCodeFile{{Path: "a.go", Content: string([]byte{0xff})}})
	require.Error(t, err)
	_, err = NormalizeLearnerFiles([]LearnerCodeFile{{Path: "a.go", Content: "package main\x00"}})
	require.Error(t, err)
	_, err = NormalizeLearnerFiles([]LearnerCodeFile{{Path: "a.go", Content: strings.Repeat("x", MaxLearnerFileBytes+1)}})
	require.Error(t, err)
	_, err = NormalizeLearnerFiles([]LearnerCodeFile{{Path: "go.mod", Content: "module example.com/answer\n"}})
	require.Error(t, err, "a module file alone is not learner code")
	_, err = NormalizeLearnerFiles([]LearnerCodeFile{{Path: "a.go", Content: "package main"}, {Path: "a.go/b.go", Content: "package main"}})
	require.Error(t, err, "one path cannot be both source file and directory")
}

func TestLearnerArtifactSnapshotBindsSavedPracticeAndCannotClaimVerification(t *testing.T) {
	artifact := LearnerArtifact{Format: LearnerArtifactFormat, WorkspaceID: uuid.NewString(), ObjectiveID: uuid.NewString(), AttemptID: uuid.NewString(), SessionID: uuid.NewString(), RevisionID: uuid.NewString(), TaskID: "practice-reproduce", PracticeContentHash: "sha256:" + strings.Repeat("a", 64), Skill: LearningTaskReproduce, SubmittedAt: time.Now().UTC(), Files: []LearnerCodeFile{{Path: "main.go", Content: "package main\n"}}}
	hash, err := LearnerArtifactHash(artifact)
	require.NoError(t, err)
	artifact.SnapshotHash = hash
	require.NoError(t, artifact.Validate())
	artifact.Files[0].Content += "// changed\n"
	require.Error(t, artifact.Validate())
	artifact.Files[0].Content = "package main\n"
	artifact.AttemptID = uuid.NewString()
	require.Error(t, artifact.Validate())
	artifact.AttemptID = uuid.Nil.String()
	_, err = LearnerArtifactHash(artifact)
	require.Error(t, err)
}
