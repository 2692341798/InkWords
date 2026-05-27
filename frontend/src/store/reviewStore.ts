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
  todayCard: ReviewCardResponse | null
  randomCard: ReviewCardResponse | null
  noteOptions: ReviewNoteOption[]
  notesPagination: Omit<ListNotesResponse, 'items'>
  currentSession: ReviewSessionResponse | null
  historyItems: ReviewHistoryItem[]
  selectedMode: ReviewMode
  latestStageFeedback: string | null
  latestHint: string | null
  finalFeedback: FinalFeedback | null
  isLoadingToday: boolean
  isLoadingRandom: boolean
  isLoadingNotes: boolean
  isLoadingHistory: boolean
  loadToday: () => Promise<void>
  loadRandom: () => Promise<void>
  loadNotes: (query?: string) => Promise<void>
  loadHistory: (limit?: number) => Promise<void>
  setSelectedMode: (mode: ReviewMode) => void
  setCurrentSession: (session: ReviewSessionResponse | null) => void
  setLatestStageFeedback: (feedback: string | null) => void
  setLatestHint: (hint: string | null) => void
  setFinalFeedback: (feedback: FinalFeedback | null) => void
  reset: () => void
}

const initialPagination = {
  total: 0,
  page: 1,
  page_size: 20,
}

// Why: 复习入口会在多个页面和交互阶段之间共享，单独 store 可以避免把 review 状态塞进 blog/generator 现有链路里。
export const useReviewStore = create<ReviewState>((set) => ({
  todayCard: null,
  randomCard: null,
  noteOptions: [],
  notesPagination: initialPagination,
  currentSession: null,
  historyItems: [],
  selectedMode: 'light_recall',
  latestStageFeedback: null,
  latestHint: null,
  finalFeedback: null,
  isLoadingToday: false,
  isLoadingRandom: false,
  isLoadingNotes: false,
  isLoadingHistory: false,

  loadToday: async () => {
    set({ isLoadingToday: true })
    try {
      const todayCard = await reviewService.getToday()
      set({ todayCard })
    } finally {
      set({ isLoadingToday: false })
    }
  },

  loadRandom: async () => {
    set({ isLoadingRandom: true })
    try {
      const randomCard = await reviewService.pickRandom()
      set({ randomCard })
    } finally {
      set({ isLoadingRandom: false })
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

  setCurrentSession: (session) => set({ currentSession: session }),

  setLatestStageFeedback: (feedback) => set({ latestStageFeedback: feedback }),

  setLatestHint: (hint) => set({ latestHint: hint }),

  setFinalFeedback: (feedback) => set({ finalFeedback: feedback }),

  reset: () =>
    set({
      todayCard: null,
      randomCard: null,
      noteOptions: [],
      notesPagination: initialPagination,
      currentSession: null,
      historyItems: [],
      selectedMode: 'light_recall',
      latestStageFeedback: null,
      latestHint: null,
      finalFeedback: null,
      isLoadingToday: false,
      isLoadingRandom: false,
      isLoadingNotes: false,
      isLoadingHistory: false,
    }),
}))
