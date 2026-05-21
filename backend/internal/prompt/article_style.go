package prompt

type ArticleStyle string

const (
	ArticleStyleGeneral          ArticleStyle = "general"
	ArticleStyleBeginnerTutorial ArticleStyle = "beginner_tutorial"
	ArticleStyleExamReview       ArticleStyle = "exam_review"
)

func (s ArticleStyle) IsValid() bool {
	switch s {
	case ArticleStyleGeneral, ArticleStyleBeginnerTutorial, ArticleStyleExamReview:
		return true
	default:
		return false
	}
}

