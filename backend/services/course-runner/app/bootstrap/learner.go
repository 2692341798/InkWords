package bootstrap

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	verification "inkwords-backend/services/course-runner/domain/verification"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func configureLearnerRuntime(enabled bool, imageDigest, sandboxProfilePath string) (sharedtextbook.LearnerVerificationCapability, *verification.BubblewrapExecutor) {
	if !enabled {
		return learnerCapability(nil, errors.New("学习者代码隔离执行未启用")), nil
	}
	sandboxProfileDigest, err := fileSHA256Digest(sandboxProfilePath)
	if err != nil {
		return learnerCapability(nil, errors.New("学习者沙箱策略不可读")), nil
	}
	identity := sharedtextbook.LearnerRunnerIdentity{ImageDigest: strings.TrimSpace(imageDigest), ToolchainVersion: runtime.Version(), SandboxProfileDigest: sandboxProfileDigest}
	if err := identity.Validate(); err != nil {
		return learnerCapability(nil, errors.New("学习者运行器身份未配置")), nil
	}
	binary, err := exec.LookPath("bwrap")
	if err != nil {
		return learnerCapability(&identity, errors.New("Bubblewrap 不可用")), nil
	}
	executor := verification.BubblewrapExecutor{Binary: binary, MaxOutputBytes: sharedtextbook.MaxLearnerVerificationOutputBytes}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	if err := executor.Preflight(ctx); err != nil {
		return learnerCapability(&identity, fmt.Errorf("隔离预检失败：%w", err)), nil
	}
	if err := executor.PreflightLearnerGoTest(ctx); err != nil {
		return learnerCapability(&identity, fmt.Errorf("固定 Go 测试预检失败：%w", err)), nil
	}
	return learnerCapability(&identity, nil), &executor
}

func fileSHA256Digest(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("sandbox profile path is empty")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(content)), nil
}

func learnerCapability(identity *sharedtextbook.LearnerRunnerIdentity, isolationErr error) sharedtextbook.LearnerVerificationCapability {
	capability := sharedtextbook.LearnerVerificationCapability{Format: sharedtextbook.LearnerVerificationCapabilityFormat, Accepted: true, Profile: sharedtextbook.LearnerGoTestProfile, Runner: identity}
	if isolationErr == nil && identity != nil {
		capability.Available = true
		return capability
	}
	if isolationErr != nil {
		capability.Reason = isolationErr.Error()
	} else {
		capability.Reason = "学习者代码隔离执行不可用"
	}
	return capability
}

func registerLearnerCapabilityRoute(r *gin.Engine, capability sharedtextbook.LearnerVerificationCapability) {
	r.GET("/internal/v1/learner-verification/capability", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 200, "data": capability})
	})
}
