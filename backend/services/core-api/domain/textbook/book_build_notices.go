package textbook

import shared "inkwords-backend/shared/kernel/textbook"

func validateBookNoticeSubjects(notices []shared.PublicationNotice, chapters []bookBuildChapterSource, artifacts []bookBuildArtifactSource) error {
	subjects := map[string]bool{}
	for _, chapter := range chapters {
		subjects["chapter-revision:"+chapter.RevisionID.String()] = true
	}
	for _, artifact := range artifacts {
		subjects["code-artifact:"+artifact.ID.String()] = true
	}
	for _, notice := range notices {
		for _, subject := range notice.SubjectRefs {
			if !subjects[subject] {
				return ErrInvalidState
			}
		}
	}
	return nil
}
