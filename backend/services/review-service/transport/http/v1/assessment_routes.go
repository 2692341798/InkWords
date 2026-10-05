package v1

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "inkwords-backend/services/review-service/app/masteryassessment"
	"inkwords-backend/shared/kernel/httpx"
)

type assessmentApplication interface {
	Preview(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (app.Preview, error)
	Start(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, app.StartInput) (app.Job, error)
	Latest(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*app.Job, error)
	Read(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (app.Job, error)
	Cancel(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (app.Job, error)
	Correct(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, app.CorrectionInput) (app.Job, error)
	Apply(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, app.ApplyInput) (app.Job, error)
}

// RegisterAssessmentRoutes keeps model execution behind an explicit POST with
// exact preview hashes. Read/preview routes never launch a provider call.
func RegisterAssessmentRoutes(r *gin.Engine, middleware gin.HandlerFunc, service assessmentApplication) {
	group := r.Group("/api/v1/mastery/objectives/:id", middleware)
	group.GET("/attempts/:attemptID/assessment-preview", func(c *gin.Context) {
		owner, objective, attempt, ok := assessmentIDs(c, "attemptID")
		if !ok {
			return
		}
		result, err := service.Preview(c.Request.Context(), owner, objective, attempt)
		assessmentResponse(c, result, err)
	})
	group.GET("/attempts/:attemptID/assessment", func(c *gin.Context) {
		owner, objective, attempt, ok := assessmentIDs(c, "attemptID")
		if !ok {
			return
		}
		result, err := service.Latest(c.Request.Context(), owner, objective, attempt)
		assessmentResponse(c, result, err)
	})
	group.POST("/attempts/:attemptID/assessments", func(c *gin.Context) {
		owner, objective, attempt, ok := assessmentIDs(c, "attemptID")
		if !ok {
			return
		}
		var input app.StartInput
		if !assessmentBody(c, &input) {
			return
		}
		result, err := service.Start(c.Request.Context(), owner, objective, attempt, input)
		assessmentResponse(c, result, err)
	})
	group.GET("/assessments/:jobID", func(c *gin.Context) {
		owner, objective, job, ok := assessmentIDs(c, "jobID")
		if !ok {
			return
		}
		result, err := service.Read(c.Request.Context(), owner, objective, job)
		assessmentResponse(c, result, err)
	})
	group.POST("/assessments/:jobID/cancel", func(c *gin.Context) {
		owner, objective, job, ok := assessmentIDs(c, "jobID")
		if !ok {
			return
		}
		result, err := service.Cancel(c.Request.Context(), owner, objective, job)
		assessmentResponse(c, result, err)
	})
	group.POST("/assessments/:jobID/corrections", func(c *gin.Context) {
		owner, objective, job, ok := assessmentIDs(c, "jobID")
		if !ok {
			return
		}
		var input app.CorrectionInput
		if !assessmentBody(c, &input) {
			return
		}
		result, err := service.Correct(c.Request.Context(), owner, objective, job, input)
		assessmentResponse(c, result, err)
	})
	group.POST("/assessments/:jobID/apply", func(c *gin.Context) {
		owner, objective, job, ok := assessmentIDs(c, "jobID")
		if !ok {
			return
		}
		var input app.ApplyInput
		if !assessmentBody(c, &input) {
			return
		}
		result, err := service.Apply(c.Request.Context(), owner, objective, job, input)
		assessmentResponse(c, result, err)
	})
}

func assessmentIDs(c *gin.Context, name string) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	owner, err := httpx.GetLocalWorkspaceID(c)
	if err != nil || owner == uuid.Nil {
		c.JSON(503, gin.H{"code": 503, "message": "本地工作区尚未准备完成"})
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	objective, err1 := uuid.Parse(c.Param("id"))
	target, err2 := uuid.Parse(c.Param(name))
	if err1 != nil || err2 != nil || objective == uuid.Nil || target == uuid.Nil {
		c.JSON(400, gin.H{"code": 400, "message": "无效的评分记录 ID"})
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return owner, objective, target, true
}

func assessmentBody(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 512*1024)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil || decoder.Decode(new(any)) != io.EOF {
		c.JSON(400, gin.H{"code": 400, "message": "评分请求不完整或包含不支持的字段"})
		return false
	}
	return true
}

func assessmentResponse(c *gin.Context, data any, err error) {
	if err == nil {
		c.JSON(200, gin.H{"code": 200, "data": data})
		return
	}
	status, message := 400, "评分操作未完成，请保留作答后重试。"
	switch {
	case errors.Is(err, app.ErrNotFound):
		status, message = 404, app.ErrNotFound.Error()
	case errors.Is(err, app.ErrConflict):
		status, message = 409, app.ErrConflict.Error()
	case errors.Is(err, app.ErrBusy):
		status, message = 409, app.ErrBusy.Error()
	case errors.Is(err, app.ErrUnavailable):
		status, message = 503, app.ErrUnavailable.Error()
	case errors.Is(err, app.ErrBudgetExceeded):
		message = app.ErrBudgetExceeded.Error()
	}
	c.JSON(status, gin.H{"code": status, "message": message})
}
