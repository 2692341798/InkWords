package v1

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	domain "inkwords-backend/services/course-runner/domain/learnerverification"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func RegisterLearnerVerificationRoutes(r *gin.Engine, runner domain.Runner) {
	r.POST("/internal/v1/learner-verification/runs", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		var reference sharedtextbook.LearnerVerificationReference
		if decoder.Decode(&reference) != nil || decoder.Decode(new(any)) != io.EOF || reference.Validate() != nil {
			c.JSON(400, gin.H{"code": 400, "message": "无效的学习者代码验证引用"})
			return
		}
		report := runner.Verify(c.Request.Context(), reference)
		c.JSON(200, gin.H{"code": 200, "data": report})
	})
}
