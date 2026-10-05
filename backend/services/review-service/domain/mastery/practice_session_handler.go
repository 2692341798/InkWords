package mastery

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type practiceSessionApplication interface {
	BeginPracticeSession(context.Context, uuid.UUID, uuid.UUID, Skill) (PracticeSessionView, error)
	LoadPracticeSession(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (PracticeSessionView, error)
	RevealPracticeHelp(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, int) (PracticeSessionView, error)
}

func (handler *Handler) practiceSessionIDs(c *gin.Context) (uuid.UUID, uuid.UUID, practiceSessionApplication, bool) {
	workspaceID, ok := handler.workspaceID(c)
	if !ok {
		return uuid.Nil, uuid.Nil, nil, false
	}
	objectiveID, err := uuid.Parse(c.Param("id"))
	if err != nil || objectiveID == uuid.Nil {
		handler.writeError(c, http.StatusBadRequest, "无效的学习目标 ID")
		return uuid.Nil, uuid.Nil, nil, false
	}
	service, ok := handler.service.(practiceSessionApplication)
	if !ok {
		handler.writeError(c, http.StatusServiceUnavailable, "学习会话暂不可用")
		return uuid.Nil, uuid.Nil, nil, false
	}
	return workspaceID, objectiveID, service, true
}

func (handler *Handler) practiceSessionResponse(c *gin.Context, view PracticeSessionView, err error) {
	if err == nil {
		handler.writeSuccess(c, view)
		return
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		handler.writeError(c, http.StatusNotFound, "学习会话不存在或不可访问")
	case errors.Is(err, ErrPracticeSessionClosed):
		handler.writeError(c, http.StatusConflict, "本次练习已记录，不能改写或新增帮助；可重新打开查看原记录。")
	default:
		handler.writeError(c, http.StatusBadRequest, "无法读取或更新学习会话，请保留作答后重试。")
	}
}

// BeginPracticeSession is an idempotent open/resume operation, not a new attempt.
func (handler *Handler) BeginPracticeSession(c *gin.Context) {
	workspaceID, objectiveID, service, ok := handler.practiceSessionIDs(c)
	if !ok {
		return
	}
	var request struct {
		Skill Skill `json:"skill"`
	}
	if c.ShouldBindJSON(&request) != nil {
		handler.writeError(c, http.StatusBadRequest, "无效的练习类型")
		return
	}
	view, err := service.BeginPracticeSession(c.Request.Context(), workspaceID, objectiveID, request.Skill)
	handler.practiceSessionResponse(c, view, err)
}

// LoadPracticeSession also recovers a committed response after a lost HTTP reply.
func (handler *Handler) LoadPracticeSession(c *gin.Context) {
	workspaceID, objectiveID, service, ok := handler.practiceSessionIDs(c)
	if !ok {
		return
	}
	sessionID, err := uuid.Parse(c.Param("sessionID"))
	if err != nil {
		handler.writeError(c, http.StatusBadRequest, "无效的练习会话 ID")
		return
	}
	view, err := service.LoadPracticeSession(c.Request.Context(), workspaceID, objectiveID, sessionID)
	handler.practiceSessionResponse(c, view, err)
}

// RevealPracticeHelp records disclosure before returning permission to display it.
func (handler *Handler) RevealPracticeHelp(c *gin.Context) {
	workspaceID, objectiveID, service, ok := handler.practiceSessionIDs(c)
	if !ok {
		return
	}
	sessionID, err := uuid.Parse(c.Param("sessionID"))
	if err != nil {
		handler.writeError(c, http.StatusBadRequest, "无效的练习会话 ID")
		return
	}
	var request struct {
		Kind  string `json:"kind"`
		Level int    `json:"level"`
	}
	if c.ShouldBindJSON(&request) != nil {
		handler.writeError(c, http.StatusBadRequest, "无效的帮助请求")
		return
	}
	view, err := service.RevealPracticeHelp(c.Request.Context(), workspaceID, objectiveID, sessionID, request.Kind, request.Level)
	handler.practiceSessionResponse(c, view, err)
}
