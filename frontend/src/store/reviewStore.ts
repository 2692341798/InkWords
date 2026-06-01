import { create } from 'zustand'
import {
  type FinalFeedback,
  type ReviewHistoryItem,
  reviewService,
  type ListNotesResponse,
  type ReviewCardResponse,
  type ReviewMode,
  type ReviewNoteOption,
  type ReviewSessionResponse,
} from '@/services/review'

interface ReviewState {
  recommendationCard: ReviewCardResponse | null
  noteOptions: ReviewNoteOption[]
  notesPagination: Omit<ListNotesResponse, 'items'>
  currentSession: ReviewSessionResponse | null
  shouldResumeSessionOnOpen: boolean
  historyItems: ReviewHistoryItem[]
  selectedMode: ReviewMode
  latestStageFeedback: string | null
  latestHint: string | null
  finalFeedback: FinalFeedback | null
  isLoadingRecommendation: boolean
  isLoadingNotes: boolean
  isLoadingHistory: boolean
  loadRecommendation: () => Promise<void>
  refreshRecommendation: () => Promise<void>
  loadNotes: (query?: string) => Promise<void>
  loadHistory: (limit?: number) => Promise<void>
  setSelectedMode: (mode: ReviewMode) => void
  setShouldResumeSessionOnOpen: (shouldResume: boolean) => void
  setCurrentSession: (session: ReviewSessionResponse | null) => void
  setShouldResumeSessionOnOpen: (shouldResume: boolean) => void
  clearSessionState: () => void
  setLatestStageFeedback: (feedback: string | null) => void
  setLatestHint: (hint: string | null) => void
  setFinalFeedback: (feedback: FinalFeedback | null) => void
  clearSessionState: () => void
  reset: () => void
}

const initialPagination = {
  total: 0,
  page: 1,
  page_size: 20,
}

// Why: 复习入口会在多个页面和交互阶段之间共享，单独 store 可以避免把 review 状态塞进 blog/generator 现有链路里。
export const useReviewStore = create<ReviewState>((set, get) => ({
  recommendationCard: null,
  noteOptions: [],
  notesPagination: initialPagination,
  currentSession: null,
  shouldResumeSessionOnOpen: false,
  historyItems: [],
  selectedMode: 'light_recall',
  latestStageFeedback: null,
  latestHint: null,
  finalFeedback: null,
  isLoadingRecommendation: false,
  isLoadingNotes: false,
  isLoadingHistory: false,

  loadRecommendation: async () => {
    set({ isLoadingRecommendation: true })
    try {
      const recommendationCard = await reviewService.pickRandom()
      set({ recommendationCard })
    } finally {
      set({ isLoadingRecommendation: false })
    }
  },

  refreshRecommendation: async () => {
    set({ isLoadingRecommendation: true })
    try {
      const currentRecommendation = get().recommendationCard
      let nextRecommendation: ReviewCardResponse | null = null

      // Why: 随机接口本身不知道当前 UI 正在展示哪一篇，所以前端最多重试几次，
      // 尽量帮用户换到另一篇内容，而不是第一次命中相同文章就直接停住。
      for (let attempt = 0; attempt < 3; attempt += 1) {
        const candidate = await reviewService.pickRandom()
        if (!currentRecommendation || candidate.note_path !== currentRecommendation.note_path) {
          nextRecommendation = candidate
          break
        }
      }

      set({ recommendationCard: nextRecommendation ?? currentRecommendation })
    } finally {
      set({ isLoadingRecommendation: false })
    }
  },

  loadNotes: async (query) => {
    set({ isLoadingNotes: true })
    try {
      const response = await reviewService.listNotes({ query })
      set({
        noteOptions: response.items,
        notesPagination: {
          total: response.total,
          page: response.page,
          page_size: response.page_size,
        },
      })
    } finally {
      set({ isLoadingNotes: false })
    }
  },

  loadHistory: async (limit = 5) => {
    set({ isLoadingHistory: true })
    try {
      const response = await reviewService.getHistory(limit)
      set({ historyItems: response.items })
    } finally {
      set({ isLoadingHistory: false })
    }
  },

  setSelectedMode: (mode) => set({ selectedMode: mode }),

  setShouldResumeSessionOnOpen: (shouldResume) => set({ shouldResumeSessionOnOpen: shouldResume }),

  setCurrentSession: (session) => set({ currentSession: session }),

  setShouldResumeSessionOnOpen: (shouldResume) => set({ shouldResumeSessionOnOpen: shouldResume }),

  clearSessionState: () =>
    set({
      currentSession: null,
      shouldResumeSessionOnOpen: false,
      latestStageFeedback: null,
      latestHint: null,
      finalFeedback: null,
    }),

  setLatestStageFeedback: (feedback) => set({ latestStageFeedback: feedback }),

  setLatestHint: (hint) => set({ latestHint: hint }),

  setFinalFeedback: (feedback) => set({ finalFeedback: feedback }),

  clearSessionState: () =>
    set({
      currentSession: null,
      shouldResumeSessionOnOpen: false,
      latestStageFeedback: null,
      latestHint: null,
      finalFeedback: null,
    }),

  reset: () =>
    set({
      recommendationCard: null,
      noteOptions: [],
      notesPagination: initialPagination,
      currentSession: null,
      shouldResumeSessionOnOpen: false,
      historyItems: [],
      selectedMode: 'light_recall',
      latestStageFeedback: null,
      latestHint: null,
      finalFeedback: null,
      isLoadingRecommendation: false,
      isLoadingNotes: false,
      isLoadingHistory: false,
    }),
}))
