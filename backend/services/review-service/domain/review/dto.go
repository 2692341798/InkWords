package review

import (
	"time"

	"github.com/google/uuid"
)

const (
	defaultReviewCardEstimatedMinutes = 5
	defaultListNotesPage              = 1
	defaultListNotesPageSize          = 20
	maxListNotesPageSize              = 100
	maxDetailedQARounds               = 3
)

// ReviewCardResponse 表示 today 与 pick 入口返回的题卡信息。
type ReviewCardResponse struct {
	NotePath         string   `json:"note_path"`
	Title            string   `json:"title"`
	SourceTitle      string   `json:"source_title"`
	ReviewReason     string   `json:"review_reason"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	AvailableModes   []string `json:"available_modes"`
}

// ListNotesQuery 描述手动选文入口的查询条件。
type ListNotesQuery struct {
	Query       string
	SeriesTitle string
	Page        int
	PageSize    int
}

// ListNotesItem 表示可手动选择的一篇候选笔记。
type ListNotesItem struct {
	NotePath       string     `json:"note_path"`
	Title          string     `json:"title"`
	SourceTitle    string     `json:"source_title"`
	LastReviewedAt *time.Time `json:"last_reviewed_at,omitempty"`
	PreferredMode  string     `json:"preferred_mode"`
}

// ListNotesResponse 表示手动选文接口的分页结果。
type ListNotesResponse struct {
	Items    []ListNotesItem `json:"items"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// ReviewHistoryItem 表示最近一次复习记录列表中的单项摘要。
type ReviewHistoryItem struct {
	SessionID   uuid.UUID  `json:"session_id"`
	NotePath    string     `json:"note_path"`
	Title       string     `json:"title"`
	SourceTitle string     `json:"source_title"`
	Mode        string     `json:"mode"`
	Status      string     `json:"status"`
	Summary     string     `json:"summary"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
}

// ReviewHistoryResponse 表示最近复习记录列表。
type ReviewHistoryResponse struct {
	Items []ReviewHistoryItem `json:"items"`
	Limit int                 `json:"limit"`
}

// CreateSessionRequest 描述创建复习会话所需的参数。
type CreateSessionRequest struct {
	NotePath  string `json:"note_path"`
	Mode      string `json:"mode"`
	EntryType string `json:"entry_type"`
}

// ReviewTurnResponse 表示对外返回的一条复习轮次记录。
type ReviewTurnResponse struct {
	TurnIndex int    `json:"turn_index"`
	Role      string `json:"role"`
	TurnType  string `json:"turn_type"`
	Content   string `json:"content"`
}

// SessionOutline 表示基于文章正文提炼出的复习快照。
type SessionOutline struct {
	Summary          string   `json:"summary"`
	MainQuestion     string   `json:"main_question"`
	CoreConcepts     []string `json:"core_concepts"`
	ProcessSteps     []string `json:"process_steps"`
	ApplicationCases []string `json:"application_cases"`
	Checkpoints      []string `json:"checkpoints"`
}

// ReviewFeedback 表示一轮回答与文章关键点的对照结果。
type ReviewFeedback struct {
	Judgement    string   `json:"judgement"`
	HitPoints    []string `json:"hit_points"`
	MissedPoints []string `json:"missed_points"`
	Suggestion   string   `json:"suggestion"`
}

// ReviewSessionResponse 表示复习会话详情。
type ReviewSessionResponse struct {
	SessionID            uuid.UUID            `json:"session_id"`
	Status               string               `json:"status"`
	Mode                 string               `json:"mode"`
	Title                string               `json:"title"`
	SourceTitle          string               `json:"source_title"`
	SourcePreview        string               `json:"source_preview"`
	ReadyToAnswer        bool                 `json:"ready_to_answer"`
	OpeningPrompt        string               `json:"opening_prompt"`
	InitialHints         []string             `json:"initial_hints"`
	SessionOutline       SessionOutline       `json:"session_outline"`
	CurrentRoundGoal     string               `json:"current_round_goal,omitempty"`
	LatestReviewFeedback *ReviewFeedback      `json:"latest_review_feedback,omitempty"`
	NextQuestion         string               `json:"next_question,omitempty"`
	TurnIndex            int                  `json:"turn_index"`
	Turns                []ReviewTurnResponse `json:"turns,omitempty"`
}

// RespondRequest 描述用户提交的一轮回答。
type RespondRequest struct {
	Answer string `json:"answer"`
}

// FinalFeedback 表示一次复习结束后的结构化反馈。
type FinalFeedback struct {
	Summary   string   `json:"summary"`
	Strengths []string `json:"strengths"`
	Gaps      []string `json:"gaps"`
	NextFocus []string `json:"next_focus"`
}

// RespondResponse 表示回答后的状态推进结果。
type RespondResponse struct {
	SessionID        uuid.UUID      `json:"session_id"`
	SessionStatus    string         `json:"session_status"`
	TurnIndex        int            `json:"turn_index"`
	StageFeedback    string         `json:"stage_feedback,omitempty"`
	CurrentRoundGoal string         `json:"current_round_goal,omitempty"`
	ReviewFeedback   ReviewFeedback `json:"review_feedback"`
	NextQuestion     string         `json:"next_question,omitempty"`
	HintText         string         `json:"hint_text,omitempty"`
	ExcerptText      string         `json:"excerpt_text,omitempty"`
	Completed        bool           `json:"completed"`
	FinalFeedback    FinalFeedback  `json:"final_feedback"`
}

// HintResponse 表示请求提示后的结果。
type HintResponse struct {
	SessionID          uuid.UUID `json:"session_id"`
	HintText           string    `json:"hint_text"`
	RemainingHintCount int       `json:"remaining_hint_count"`
}

// FinishResponse 表示显式结束训练后的最终反馈。
type FinishResponse struct {
	SessionID     uuid.UUID     `json:"session_id"`
	SessionStatus string        `json:"session_status"`
	FinalFeedback FinalFeedback `json:"final_feedback"`
}
