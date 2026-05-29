# Review Entry Merge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge the duplicated automatic review entry cards into one recommendation card, make refresh rotate to a different article when possible, and fix backend random picking so it is truly random.

**Architecture:** Keep the existing review session flow intact, but collapse the front-end automatic entry state into one displayed recommendation card. Use the stable `today` API for first load, and use a corrected backend random picker for refresh behavior when the UI needs a different article than the current one.

**Tech Stack:** React 18, Vite, Zustand, Vitest, Go 1.21, Gin, Testify

---

### Task 1: Fix Backend Random Picker

**Files:**
- Modify: `backend/internal/domain/review/picker.go`
- Modify: `backend/internal/domain/review/picker_test.go`

- [ ] **Step 1: Write the failing picker test**

```go
func TestPicker_PickRandom_UsesRandomIndexAmongEligibleItems(t *testing.T) {
	t.Parallel()

	notes := []ReviewNote{
		{NotePath: "wiki/concepts/a.md", Title: "A"},
		{NotePath: "wiki/concepts/b.md", Title: "B"},
		{NotePath: "wiki/concepts/c.md", Title: "C"},
	}

	got := pickRandomWithRand(notes, map[string]bool{
		"wiki/concepts/a.md": true,
	}, rand.New(rand.NewSource(2)))

	require.NotEqual(t, "wiki/concepts/a.md", got.NotePath)
	require.Contains(t, []string{"wiki/concepts/b.md", "wiki/concepts/c.md"}, got.NotePath)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/review -run TestPicker_PickRandom_UsesRandomIndexAmongEligibleItems`
Expected: FAIL because `pickRandomWithRand` does not exist or `PickRandom` still behaves deterministically.

- [ ] **Step 3: Write minimal implementation**

```go
func PickRandom(notes []ReviewNote, recent map[string]bool) ReviewNote {
	return pickRandomWithRand(notes, recent, rand.New(rand.NewSource(time.Now().UnixNano())))
}

func pickRandomWithRand(notes []ReviewNote, recent map[string]bool, rng *rand.Rand) ReviewNote {
	if len(notes) == 0 {
		return ReviewNote{}
	}

	candidates := make([]ReviewNote, 0, len(notes))
	for _, note := range notes {
		if !recent[note.NotePath] {
			candidates = append(candidates, note)
		}
	}

	if len(candidates) == 0 {
		candidates = notes
	}

	return candidates[rng.Intn(len(candidates))]
}
```

- [ ] **Step 4: Run backend review tests**

Run: `go test ./internal/domain/review/...`
Expected: PASS for picker and service tests.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/domain/review/picker.go backend/internal/domain/review/picker_test.go
git commit -m "fix(review): randomize review pick selection"
```

### Task 2: Collapse Frontend Entry State Into One Recommendation Card

**Files:**
- Modify: `frontend/src/store/reviewStore.ts`
- Modify: `frontend/src/store/reviewStore.test.ts`
- Modify: `frontend/src/components/review/ReviewEntryCards.tsx`

- [ ] **Step 1: Write failing store tests for merged recommendation state**

```ts
it('loads the recommendation card from today by default', async () => {
  mockedReviewService.getToday.mockResolvedValue(todayCard)

  await useReviewStore.getState().loadRecommendation()

  expect(useReviewStore.getState().recommendationCard?.note_path).toBe(todayCard.note_path)
})

it('refreshes the recommendation card with a different random article when available', async () => {
  mockedReviewService.pickRandom.mockResolvedValue(randomCard)
  useReviewStore.setState({ recommendationCard: todayCard })

  await useReviewStore.getState().refreshRecommendation()

  expect(useReviewStore.getState().recommendationCard?.note_path).toBe(randomCard.note_path)
})
```

- [ ] **Step 2: Run store tests to verify they fail**

Run: `cd frontend && npm run test -- reviewStore.test.ts`
Expected: FAIL because `recommendationCard`, `loadRecommendation`, and `refreshRecommendation` do not exist.

- [ ] **Step 3: Write minimal store implementation**

```ts
interface ReviewState {
  recommendationCard: ReviewCardResponse | null
  isLoadingRecommendation: boolean
  loadRecommendation: () => Promise<void>
  refreshRecommendation: () => Promise<void>
}

loadRecommendation: async () => {
  set({ isLoadingRecommendation: true })
  try {
    const recommendationCard = await reviewService.getToday()
    set({ recommendationCard })
  } finally {
    set({ isLoadingRecommendation: false })
  }
},

refreshRecommendation: async () => {
  set({ isLoadingRecommendation: true })
  try {
    const current = get().recommendationCard
    const nextCard = await reviewService.pickRandom()
    set({
      recommendationCard:
        current && nextCard.note_path === current.note_path ? current : nextCard,
    })
  } finally {
    set({ isLoadingRecommendation: false })
  }
},
```

- [ ] **Step 4: Update the entry card component to render one automatic card**

```tsx
<ReviewCard
  title="推荐一篇"
  description="先用系统推荐进入状态，刷新时会尽量换一篇不同内容。"
  icon={Sparkles}
  detail={props.recommendationCard}
  loading={props.isLoadingRecommendation}
  actionLabel="用这篇开始"
  refreshLabel="换一篇"
  onAction={props.onStartRecommendation}
  onRefresh={props.onRefreshRecommendation}
/>
```

- [ ] **Step 5: Run store and component tests**

Run: `cd frontend && npm run test -- reviewStore.test.ts`
Expected: PASS with merged recommendation state.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/store/reviewStore.ts frontend/src/store/reviewStore.test.ts frontend/src/components/review/ReviewEntryCards.tsx
git commit -m "refactor(review): merge duplicate review entry cards"
```

### Task 3: Wire Knowledge Review Page To The Merged Recommendation Flow

**Files:**
- Modify: `frontend/src/pages/KnowledgeReview.tsx`
- Modify: `frontend/src/hooks/useKnowledgeReview.ts`

- [ ] **Step 1: Write the failing page-level test or assertion update**

```ts
expect(screen.getByText('推荐一篇')).toBeInTheDocument()
expect(screen.queryByText('随机抽一篇')).not.toBeInTheDocument()
expect(screen.queryByText('开始今日复习')).not.toBeInTheDocument()
```

- [ ] **Step 2: Run the targeted page test**

Run: `cd frontend && npm run test -- KnowledgeReview`
Expected: FAIL because the page still renders the old two-card layout.

- [ ] **Step 3: Replace old page wiring with recommendation-only handlers**

```tsx
<ReviewEntryCards
  recommendationCard={reviewStore.recommendationCard}
  isLoadingRecommendation={reviewStore.isLoadingRecommendation}
  onRefreshRecommendation={() => reviewStore.refreshRecommendation()}
  onStartRecommendation={async () => {
    if (!reviewStore.recommendationCard) {
      await reviewStore.loadRecommendation()
    }
    const card = useReviewStore.getState().recommendationCard
    if (card) {
      await startSession(card, 'today')
    }
  }}
  onOpenPicker={async () => {
    setIsPickerOpen(true)
    await reviewStore.loadNotes()
  }}
/>
```

- [ ] **Step 4: Update summary copy to match the new two-entry experience**

```tsx
const currentEntrySummary = reviewStore.currentSession
  ? reviewStore.currentSession.title
  : effectiveIsPickerOpen
    ? 'Hand-picked article'
    : 'Recommended article / manual pick'
```

- [ ] **Step 5: Run page and hook-adjacent tests**

Run: `cd frontend && npm run test -- KnowledgeReview reviewStore.test.ts`
Expected: PASS and no references to removed random-card state remain.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/pages/KnowledgeReview.tsx frontend/src/hooks/useKnowledgeReview.ts
git commit -m "fix(review): wire merged recommendation entry flow"
```

### Task 4: Validation And Documentation Sync

**Files:**
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_PRD.md`

- [ ] **Step 1: Run full targeted validation**

Run: `cd frontend && npm run test -- reviewStore.test.ts review.test.ts`
Expected: PASS

Run: `cd backend && go test ./internal/domain/review/...`
Expected: PASS

- [ ] **Step 2: Run diagnostics on edited frontend files**

Run: use diagnostics tooling on the modified React files
Expected: no new type or lint errors

- [ ] **Step 3: Update docs to reflect merged entry behavior**

```md
- Review entry now exposes one recommendation card plus one manual-pick card.
- Recommendation loads from `/api/v1/review/today`.
- Refresh rotates with `/api/v1/review/pick` and now depends on real random selection.
```

- [ ] **Step 4: Review git diff before commit or PR**

Run: `git diff -- docs/superpowers/specs/2026-05-29-review-entry-merge-design.md docs/superpowers/plans/2026-05-29-review-entry-merge-plan.md`
Expected: plan and design remain aligned with the implemented scope.

- [ ] **Step 5: Commit**

```bash
git add README.md .trae/documents/InkWords_API.md .trae/documents/InkWords_Architecture.md .trae/documents/InkWords_Conversation_Log.md .trae/documents/InkWords_Development_Plan_and_Log.md .trae/documents/InkWords_PRD.md
git commit -m "docs(review): sync merged recommendation entry behavior"
```
