package mastery

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type learnerArtifactReader interface {
	GetLearnerArtifact(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*sharedtextbook.LearnerArtifact, error)
}

// LearnerArtifact returns exact submitted files; fetching never executes them.
func (handler *Handler) LearnerArtifact(c *gin.Context) {
	owner, ok := handler.workspaceID(c)
	if !ok {
		return
	}
	objective, err1 := uuid.Parse(c.Param("id"))
	attempt, err2 := uuid.Parse(c.Param("attemptID"))
	if err1 != nil || err2 != nil || objective == uuid.Nil || attempt == uuid.Nil {
		handler.writeError(c, 400, "无效的学习作答 ID")
		return
	}
	reader, ok := handler.service.(learnerArtifactReader)
	if !ok {
		handler.writeError(c, 503, "代码快照读取暂不可用")
		return
	}
	artifact, err := reader.GetLearnerArtifact(c.Request.Context(), owner, objective, attempt)
	if err != nil {
		handler.writeError(c, 404, "这份作答的代码快照不可用")
		return
	}
	handler.writeSuccess(c, artifact)
}
