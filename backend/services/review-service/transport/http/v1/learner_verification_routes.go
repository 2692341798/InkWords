package v1

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "inkwords-backend/services/review-service/app/masteryverification"
	"inkwords-backend/shared/kernel/httpx"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type learnerVerificationApplication interface {
	Preview(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (app.Preview, error)
	Start(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, app.StartInput) (app.Job, error)
	Latest(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*app.Job, error)
	Read(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (app.Job, error)
	Cancel(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (app.Job, error)
	Resolve(context.Context, sharedtextbook.LearnerVerificationReference) (sharedtextbook.LearnerVerificationInput, error)
}

func RegisterLearnerVerificationRoutes(r *gin.Engine, middleware gin.HandlerFunc, service learnerVerificationApplication) {
	group := r.Group("/api/v1/mastery/objectives/:id", middleware)
	group.GET("/attempts/:attemptID/verification-preview", func(c *gin.Context) {
		owner, objective, attempt, ok := learnerVerificationIDs(c, "attemptID")
		if !ok {
			return
		}
		preview, err := service.Preview(c.Request.Context(), owner, objective, attempt)
		if err != nil {
			c.JSON(503, gin.H{"code": 503, "message": "学习者代码验证预检暂不可用"})
			return
		}
		c.JSON(200, gin.H{"code": 200, "data": preview})
	})
	group.GET("/attempts/:attemptID/verification", func(c *gin.Context) {
		owner, objective, attempt, ok := learnerVerificationIDs(c, "attemptID")
		if !ok {
			return
		}
		job, err := service.Latest(c.Request.Context(), owner, objective, attempt)
		learnerVerificationResponse(c, job, err)
	})
	group.POST("/attempts/:attemptID/verifications", func(c *gin.Context) {
		owner, objective, attempt, ok := learnerVerificationIDs(c, "attemptID")
		if !ok {
			return
		}
		var input app.StartInput
		if !learnerVerificationBody(c, &input) {
			return
		}
		job, err := service.Start(c.Request.Context(), owner, objective, attempt, input)
		learnerVerificationResponse(c, job, err)
	})
	group.GET("/verifications/:runID", func(c *gin.Context) {
		owner, objective, run, ok := learnerVerificationIDs(c, "runID")
		if !ok {
			return
		}
		job, err := service.Read(c.Request.Context(), owner, objective, run)
		learnerVerificationResponse(c, job, err)
	})
	group.POST("/verifications/:runID/cancel", func(c *gin.Context) {
		owner, objective, run, ok := learnerVerificationIDs(c, "runID")
		if !ok {
			return
		}
		job, err := service.Cancel(c.Request.Context(), owner, objective, run)
		learnerVerificationResponse(c, job, err)
	})
	r.POST("/internal/v1/learner-verification/resolve", func(c *gin.Context) {
		var reference sharedtextbook.LearnerVerificationReference
		if !learnerVerificationBody(c, &reference) {
			return
		}
		if reference.Validate() != nil {
			c.JSON(400, gin.H{"code": 400, "message": "无效的学习者验证引用"})
			return
		}
		input, err := service.Resolve(c.Request.Context(), reference)
		if err != nil {
			c.JSON(404, gin.H{"code": 404, "message": "学习者验证引用不可用"})
			return
		}
		c.JSON(200, gin.H{"code": 200, "data": input})
	})
}

func learnerVerificationIDs(c *gin.Context, targetName string) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	owner, err := httpx.GetLocalWorkspaceID(c)
	objective, objectiveErr := uuid.Parse(c.Param("id"))
	target, targetErr := uuid.Parse(c.Param(targetName))
	if err != nil || owner == uuid.Nil {
		c.JSON(503, gin.H{"code": 503, "message": "本地工作区尚未准备完成"})
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	if objectiveErr != nil || targetErr != nil || objective == uuid.Nil || target == uuid.Nil {
		c.JSON(400, gin.H{"code": 400, "message": "无效的学习者验证记录 ID"})
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return owner, objective, target, true
}

func learnerVerificationBody(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil || decoder.Decode(new(any)) != io.EOF {
		c.JSON(400, gin.H{"code": 400, "message": "学习者验证请求不完整或包含不支持的字段"})
		return false
	}
	return true
}

func learnerVerificationResponse(c *gin.Context, data any, err error) {
	if err == nil {
		c.JSON(200, gin.H{"code": 200, "data": data})
		return
	}
	status, message := 400, "学习者代码验证操作未完成"
	switch {
	case errors.Is(err, app.ErrNotFound):
		status, message = 404, app.ErrNotFound.Error()
	case errors.Is(err, app.ErrConflict):
		status, message = 409, app.ErrConflict.Error()
	case errors.Is(err, app.ErrBusy):
		status, message = 409, app.ErrBusy.Error()
	case errors.Is(err, app.ErrUnavailable):
		status, message = 503, app.ErrUnavailable.Error()
	}
	c.JSON(status, gin.H{"code": status, "message": message})
}
