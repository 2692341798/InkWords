package textbook

import sharedtextbook "inkwords-backend/shared/kernel/textbook"

// applyGeneratedPracticePolicy fills server-owned flags only on freshly decoded
// provider output. It never repairs task content or edits stored revisions and
// receipts; the complete practice and manuscript binding gates still run next.
func applyGeneratedPracticePolicy(set *sharedtextbook.PracticeSet) {
	for i := range set.Tasks {
		task := &set.Tasks[i]
		for _, dimension := range sharedtextbook.PracticeRubricDimensions(task.Mode) {
			for j := range task.Rubric {
				if task.Rubric[j].ID == dimension.ID {
					task.Rubric[j].RequiresRuntime = dimension.RequiresRuntime
				}
			}
		}
	}
}
