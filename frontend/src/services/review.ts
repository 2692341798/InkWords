import { requestEnvelope } from './apiClient'
import { apiRoutes } from './apiRoutes'

export type ReviewMode = 'light_recall' | 'detailed_qa'
export type ReviewEntryType = 'today' | 'manual_random' | 'manual_select'

export interface ReviewCardResponse {
  note_path: string
  title: string
  source_title: string
  review_reason: string
  estimated_minutes: number
  available_modes: ReviewMode[]
}

export interface ReviewNoteOption {
  note_path: string
  title: string
  source_title: string
  last_reviewed_at: string | null
  preferred_mode: ReviewMode
}

export interface ListNotesResponse {
  items: ReviewNoteOption[]
  total: number
  page: number
  page_size: number
}

export interface ReviewHistoryItem {
  session_id: string
  note_path: string
  title: string
  source_title: string
  mode: ReviewMode
  status: string
  summary: string
  reviewed_at: string | null
}

export interface ReviewHistoryResponse {
  items: ReviewHistoryItem[]
  limit: number
}

export interface ReviewTurnResponse {
  turn_index: number
  role: string
  turn_type: string
  content: string
}

interface SessionOutline {
  summary: string
  main_question: string
  core_concepts: string[]
  process_steps: string[]
  application_cases: string[]
  checkpoints: string[]
}

interface ReviewFeedback {
  judgement: string
  hit_points: string[]
  missed_points: string[]
  suggestion: string
}

export interface ReviewSessionResponse {
  session_id: string
  status: string
  mode: ReviewMode
  title: string
  source_title?: string
  source_preview?: string
  ready_to_answer?: boolean
  opening_prompt: string
  initial_hints: string[]
  session_outline: SessionOutline
  current_round_goal?: string
  latest_review_feedback?: ReviewFeedback | null
  next_question?: string
  turn_index: number
  turns?: ReviewTurnResponse[]
}

export interface CreateSessionRequest {
  note_path: string
  mode: ReviewMode
  entry_type: ReviewEntryType
}

export interface RespondRequest {
  answer: string
}

export interface FinalFeedback {
  summary: string
  strengths: string[]
  gaps: string[]
  next_focus: string[]
}

export interface RespondResponse {
  session_id: string
  session_status: string
  turn_index: number
  stage_feedback?: string
  current_round_goal?: string
  review_feedback: ReviewFeedback
  next_question?: string
  hint_text?: string
  excerpt_text?: string
  completed: boolean
  final_feedback: FinalFeedback
}

export interface HintResponse {
  session_id: string
  hint_text: string
  remaining_hint_count: number
}

export interface FinishResponse {
  session_id: string
  session_status: string
  final_feedback: FinalFeedback
}

interface ListNotesParams {
  query?: string
  seriesTitle?: string
  page?: number
  pageSize?: number
}

export const reviewService = {
  getToday() {
    return requestEnvelope<ReviewCardResponse>(apiRoutes.reviewService.today, {
      fallbackMessage: '请求复习接口失败',
    })
  },

  pickRandom() {
    return requestEnvelope<ReviewCardResponse>(apiRoutes.reviewService.pick, {
      method: 'POST',
      fallbackMessage: '请求复习接口失败',
    })
  },

  listNotes(params: ListNotesParams = {}) {
    const search = new URLSearchParams()
    if (params.query) {
      search.set('query', params.query)
    }
    if (params.seriesTitle) {
      search.set('series_title', params.seriesTitle)
    }
    search.set('page', String(params.page ?? 1))
    search.set('page_size', String(params.pageSize ?? 20))

    return requestEnvelope<ListNotesResponse>(`${apiRoutes.reviewService.notes}?${search.toString()}`, {
      fallbackMessage: '请求复习接口失败',
    })
  },

  getHistory(limit = 5) {
    const search = new URLSearchParams({ limit: String(limit) })
    return requestEnvelope<ReviewHistoryResponse>(`${apiRoutes.reviewService.history}?${search.toString()}`, {
      fallbackMessage: '请求复习接口失败',
    })
  },

  createSession(payload: CreateSessionRequest) {
    return requestEnvelope<ReviewSessionResponse>(apiRoutes.reviewService.sessions, {
      method: 'POST',
      json: payload,
      fallbackMessage: '请求复习接口失败',
    })
  },

  getSession(sessionId: string) {
    return requestEnvelope<ReviewSessionResponse>(apiRoutes.reviewService.session(sessionId), {
      fallbackMessage: '请求复习接口失败',
    })
  },

  respond(sessionId: string, payload: RespondRequest) {
    return requestEnvelope<RespondResponse>(apiRoutes.reviewService.respond(sessionId), {
      method: 'POST',
      json: payload,
      fallbackMessage: '请求复习接口失败',
    })
  },

  requestHint(sessionId: string) {
    return requestEnvelope<HintResponse>(apiRoutes.reviewService.hint(sessionId), {
      method: 'POST',
      json: {},
      fallbackMessage: '请求复习接口失败',
    })
  },

  finish(sessionId: string) {
    return requestEnvelope<FinishResponse>(apiRoutes.reviewService.finish(sessionId), {
      method: 'POST',
      json: {},
      fallbackMessage: '请求复习接口失败',
    })
  },
}
