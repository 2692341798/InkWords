# Review Question Entry Button Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a visible `提问开始` button to the recommendation card on the knowledge review page so users can launch the current recommended article directly in `细致提问` mode.

**Architecture:** Keep the existing review session creation flow and only extend the recommendation-card action area. The new button is wired at the page level: it sets `selectedMode` to `detailed_qa`, then reuses the current recommendation-card startup path. Existing manual picker behavior and in-session mode locking remain unchanged.

**Tech Stack:** React 18, Vite, Zustand, Vitest

---

### Task 1: Expose The New Entry Button In The Recommendation Card

**Files:**
- Modify: `frontend/src/components/review/ReviewEntryCards.tsx`
- Modify: `frontend/src/components/review/ReviewEntryCards.test.tsx`

- [ ] **Step 1: Write the failing component test**

```tsx
it('renders a dedicated question-start action on the recommendation card', () => {
  const html = renderToStaticMarkup(
    <ReviewEntryCards
      recommendationCard={{
        note_path: 'wiki/concepts/random.md',
        title: '随机文章',
        source_title: '知识库',
        review_reason: '从随机文章开始复习',
        estimated_minutes: 6,
        available_modes: ['light_recall', 'detailed_qa'],
      }}
      isLoadingRecommendation={false}
      onRefreshRecommendation={vi.fn()}
      onStartRecommendation={vi.fn()}
      onStartQuestionRecommendation={vi.fn()}
      onOpenPicker={vi.fn()}
    />,
  )

  expect(html).toContain('随机抽一篇')
  expect(html).toContain('用这篇开始')
  expect(html).toContain('提问开始')
  expect(html).toContain('再抽一篇')
})
```

- [ ] **Step 2: Run the component test to verify it fails**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- ReviewEntryCards.test.tsx`
Expected: FAIL because `onStartQuestionRecommendation` does not exist in `ReviewEntryCardsProps` and the rendered markup does not contain `提问开始`.

- [ ] **Step 3: Add the new prop and button with the minimal implementation**

```tsx
interface ReviewEntryCardsProps {
  recommendationCard: ReviewCardResponse | null
  isLoadingRecommendation: boolean
  onRefreshRecommendation: () => Promise<void> | void
  onStartRecommendation: () => Promise<void> | void
  onStartQuestionRecommendation: () => Promise<void> | void
  onOpenPicker: () => Promise<void> | void
}
```

```tsx
<div className="mt-4 flex flex-wrap gap-3">
  <Button className="flex-1" onClick={onAction} disabled={loading}>
    {actionLabel}
  </Button>
  <Button variant="outline" className="flex-1" onClick={onQuestionAction} disabled={loading}>
    {questionActionLabel}
  </Button>
  <Button variant="outline" className="flex-1" onClick={onRefresh} disabled={loading}>
    {refreshLabel}
  </Button>
</div>
```

```tsx
<ReviewCard
  title="随机抽一篇"
  description="适合快速进入状态，先从一篇随机文章开始复习。"
  icon={Shuffle}
  detail={props.recommendationCard}
  loading={props.isLoadingRecommendation}
  actionLabel="用这篇开始"
  questionActionLabel="提问开始"
  refreshLabel="再抽一篇"
  onAction={props.onStartRecommendation}
  onQuestionAction={props.onStartQuestionRecommendation}
  onRefresh={props.onRefreshRecommendation}
/>
```

- [ ] **Step 4: Run the component test to verify it passes**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- ReviewEntryCards.test.tsx`
Expected: PASS and the rendered recommendation card contains all three actions.

- [ ] **Step 5: Commit the component-only change**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/components/review/ReviewEntryCards.tsx frontend/src/components/review/ReviewEntryCards.test.tsx
git commit -m "feat(review): add question-start entry action"
```

### Task 2: Wire The New Action To Detailed QA Session Startup

**Files:**
- Modify: `frontend/src/pages/KnowledgeReview.tsx`

- [ ] **Step 1: Write the failing page test by adding a new targeted test file**

Create: `frontend/src/pages/KnowledgeReview.test.tsx`

```tsx
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { KnowledgeReview } from './KnowledgeReview'

const startSessionMock = vi.fn()
const initializeMock = vi.fn().mockResolvedValue(undefined)
const respondMock = vi.fn()
const requestHintMock = vi.fn()
const finishMock = vi.fn()
const clearSessionMock = vi.fn()

vi.mock('@/hooks/useKnowledgeReview', () => ({
  useKnowledgeReview: () => ({
    initialize: initializeMock,
    startSession: startSessionMock,
    respond: respondMock,
    requestHint: requestHintMock,
    finish: finishMock,
    clearSession: clearSessionMock,
  }),
}))

vi.mock('@/store/reviewStore', () => ({
  useReviewStore: () => ({
    recommendationCard: {
      note_path: 'wiki/concepts/random.md',
      title: '随机文章',
      source_title: '知识库',
      review_reason: '从随机文章开始复习',
      estimated_minutes: 6,
      available_modes: ['light_recall', 'detailed_qa'],
    },
    isLoadingRecommendation: false,
    currentSession: null,
    shouldResumeSessionOnOpen: false,
    latestStageFeedback: null,
    latestHint: null,
    finalFeedback: null,
    historyItems: [],
    isLoadingHistory: false,
    noteOptions: [],
    isLoadingNotes: false,
    selectedMode: 'light_recall',
    loadRecommendation: vi.fn().mockResolvedValue(undefined),
    refreshRecommendation: vi.fn().mockResolvedValue(undefined),
    loadNotes: vi.fn().mockResolvedValue(undefined),
    loadHistory: vi.fn().mockResolvedValue(undefined),
    setSelectedMode: vi.fn(),
  }),
}))

describe('KnowledgeReview', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('starts the recommended session in detailed question mode from the entry card', async () => {
    const user = userEvent.setup()
    render(<KnowledgeReview />)

    await user.click(screen.getByRole('button', { name: '提问开始' }))

    const { useReviewStore } = await import('@/store/reviewStore')
    const reviewStore = useReviewStore()
    await waitFor(() => {
      expect(reviewStore.setSelectedMode).toHaveBeenCalledWith('detailed_qa')
      expect(startSessionMock).toHaveBeenCalledWith(
        expect.objectContaining({ note_path: 'wiki/concepts/random.md' }),
        'manual_random',
      )
    })
  })
})
```

- [ ] **Step 2: Run the page test to verify it fails**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- KnowledgeReview.test.tsx`
Expected: FAIL because `提问开始` is not rendered yet and the page does not pass a question-start handler into `ReviewEntryCards`.

- [ ] **Step 3: Wire the page handler with the minimal implementation**

```tsx
<ReviewEntryCards
  recommendationCard={reviewStore.recommendationCard}
  isLoadingRecommendation={reviewStore.isLoadingRecommendation}
  onRefreshRecommendation={async () => {
    clearSession()
    await reviewStore.refreshRecommendation()
  }}
  onStartRecommendation={async () => {
    if (!reviewStore.recommendationCard) {
      await reviewStore.loadRecommendation()
    }
    const card = useReviewStore.getState().recommendationCard
    if (card) {
      await startSession(card, 'manual_random')
    }
  }}
  onStartQuestionRecommendation={async () => {
    if (!reviewStore.recommendationCard) {
      await reviewStore.loadRecommendation()
    }
    const card = useReviewStore.getState().recommendationCard
    if (card) {
      reviewStore.setSelectedMode('detailed_qa')
      await startSession(card, 'manual_random')
    }
  }}
  onOpenPicker={async () => {
    clearSession()
    setIsPickerOpen(true)
    await reviewStore.loadNotes()
  }}
/>
```

- [ ] **Step 4: Run the page and component tests to verify they pass together**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- ReviewEntryCards.test.tsx KnowledgeReview.test.tsx`
Expected: PASS and the new action starts the recommendation flow in `detailed_qa` mode.

- [ ] **Step 5: Commit the page wiring change**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/pages/KnowledgeReview.tsx frontend/src/pages/KnowledgeReview.test.tsx
git commit -m "feat(review): wire question-start recommendation flow"
```

### Task 3: Validate The Focused Frontend Change

**Files:**
- Check: `frontend/src/components/review/ReviewEntryCards.tsx`
- Check: `frontend/src/components/review/ReviewEntryCards.test.tsx`
- Check: `frontend/src/pages/KnowledgeReview.tsx`
- Check: `frontend/src/pages/KnowledgeReview.test.tsx`

- [ ] **Step 1: Run the adjacent review store regression test**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- reviewStore.test.ts`
Expected: PASS and no store behavior changed.

- [ ] **Step 2: Run the full targeted frontend test set**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- ReviewEntryCards.test.tsx KnowledgeReview.test.tsx reviewStore.test.ts`
Expected: PASS with no regressions around recommendation loading or entry-card rendering.

- [ ] **Step 3: Run diagnostics on edited frontend files**

Run: use diagnostics tooling for:
- `frontend/src/components/review/ReviewEntryCards.tsx`
- `frontend/src/pages/KnowledgeReview.tsx`
- `frontend/src/pages/KnowledgeReview.test.tsx`

Expected: no new type or lint errors.

- [ ] **Step 4: Review the scoped diff**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git diff -- frontend/src/components/review/ReviewEntryCards.tsx frontend/src/components/review/ReviewEntryCards.test.tsx frontend/src/pages/KnowledgeReview.tsx frontend/src/pages/KnowledgeReview.test.tsx
```

Expected: diff only shows the new recommendation-card question-start entry and its focused tests.

- [ ] **Step 5: Commit the validation checkpoint if needed**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git status
```

Expected: clean working tree for the planned files, or only unrelated pre-existing changes outside this feature.
