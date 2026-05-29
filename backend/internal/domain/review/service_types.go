package review

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/datatypes"

	"inkwords-backend/internal/model"
)

const recentReviewWindow = 72 * time.Hour

var (
	errNoEligibleReviewNotes = errors.New("暂无可复习的知识卡")
	errInvalidReviewMode     = errors.New("不支持的复习模式")
	errInvalidReviewEntry    = errors.New("不支持的复习入口")
	errReviewNoteNotFound    = errors.New("指定的复习笔记不存在")
	errReviewSessionNotFound = errors.New("复习会话不存在")
	errReviewSessionDenied   = errors.New("无权访问该复习会话")
	errReviewSessionClosed   = errors.New("复习会话已结束")
	errReviewHintExhausted   = errors.New("提示次数已用尽")
	errEmptyReviewAnswer     = errors.New("回答内容不能为空")
)

// NoteSource 定义 review 领域所需的笔记读取能力。
type NoteSource interface {
	ListEligibleNotes(ctx context.Context) ([]ReviewNote, error)
}

// Service 提供 today、pick、notes 三类入口能力。
type Service struct {
	repo       Repository
	noteSource NoteSource
	now        func() time.Time
}

// NewService 创建 review 领域服务。
func NewService(repo Repository, noteSource NoteSource) *Service {
	return &Service{
		repo:       repo,
		noteSource: noteSource,
		now:        time.Now,
	}
}

func resolveReviewedAt(session model.ReviewSession) time.Time {
	if session.CompletedAt != nil && !session.CompletedAt.IsZero() {
		return *session.CompletedAt
	}
	if !session.UpdatedAt.IsZero() {
		return session.UpdatedAt
	}
	if !session.StartedAt.IsZero() {
		return session.StartedAt
	}
	return session.CreatedAt
}

func nextTurnIndex(turns []model.ReviewTurn) int {
	if len(turns) == 0 {
		return 1
	}
	return turns[len(turns)-1].TurnIndex + 1
}

func countUserAnswers(turns []model.ReviewTurn) int {
	count := 0
	for _, turn := range turns {
		if turn.Role == model.ReviewTurnRoleUser && turn.TurnType == model.ReviewTurnTypeAnswer {
			count++
		}
	}
	return count
}

func lastUserAnswer(turns []model.ReviewTurn) string {
	for idx := len(turns) - 1; idx >= 0; idx-- {
		if turns[idx].Role == model.ReviewTurnRoleUser && turns[idx].TurnType == model.ReviewTurnTypeAnswer {
			return turns[idx].Content
		}
	}
	return ""
}

func toTurnResponses(turns []model.ReviewTurn) []ReviewTurnResponse {
	items := make([]ReviewTurnResponse, 0, len(turns))
	for _, turn := range turns {
		items = append(items, toTurnResponse(turn))
	}
	return items
}

func toTurnResponse(turn model.ReviewTurn) ReviewTurnResponse {
	return ReviewTurnResponse{
		TurnIndex: turn.TurnIndex,
		Role:      turn.Role,
		TurnType:  turn.TurnType,
		Content:   turn.Content,
	}
}

func isSupportedReviewMode(mode string) bool {
	switch mode {
	case model.ReviewModeLightRecall, model.ReviewModeDetailedQA:
		return true
	default:
		return false
	}
}

func isSupportedReviewEntryType(entryType string) bool {
	switch entryType {
	case model.ReviewEntryTypeToday, model.ReviewEntryTypeManualRandom, model.ReviewEntryTypeManualSelect:
		return true
	default:
		return false
	}
}

func isClosedStatus(status string) bool {
	return status == model.ReviewStatusCompleted || status == model.ReviewStatusAbandoned
}

func decodeStringSlice(raw []byte) []string {
	var values []string
	if len(raw) == 0 {
		return []string{}
	}
	if err := json.Unmarshal(raw, &values); err != nil {
		return []string{}
	}
	return values
}

func mustMarshalJSON(value any) datatypes.JSON {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return datatypes.JSON(data)
}
