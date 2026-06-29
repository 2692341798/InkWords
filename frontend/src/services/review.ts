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

interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

interface ListNotesParams {
  query?: string
  seriesTitle?: string
  page?: number
  pageSize?: number
}

const getLocalStorage = () => {
  if (typeof window === 'undefined' && typeof globalThis.localStorage === 'undefined') {
    return null
  }
  return globalThis.localStorage ?? null
}

const buildHeaders = (init?: RequestInit) => {
  const headers = new Headers(init?.headers)
  const token = getLocalStorage()?.getItem('token')

  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  if (init?.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  return headers
}

async function requestJson<T>(url: string, init?: RequestInit) {
  const response = await fetch(url, {
    ...init,
    headers: buildHeaders(init),
  })

  if (response.status === 401) {
    getLocalStorage()?.removeItem('token')
    throw new Error('登录已过期，请重新登录')
  }

  const payload = (await response.json().catch(() => null)) as ApiEnvelope<T> | null
  if (!response.ok || !payload || payload.code !== 200) {
    throw new Error(payload?.message || '请求复习接口失败')
  }

  return payload.data
}

export const reviewService = {
  getToday() {
    return requestJson<ReviewCardResponse>('/api/v1/review/today')
  },

  pickRandom() {
    return requestJson<ReviewCardResponse>('/api/v1/review/pick', {
      method: 'POST',
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

    return requestJson<ListNotesResponse>(`/api/v1/review/notes?${search.toString()}`)
  },

  getHistory(limit = 5) {
    return requestJson<ReviewHistoryResponse>(`/api/v1/review/history?limit=${limit}`)
  },

  createSession(payload: CreateSessionRequest) {
    return requestJson<ReviewSessionResponse>('/api/v1/review/sessions', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  },

  getSession(sessionId: string) {
    return requestJson<ReviewSessionResponse>(`/api/v1/review/sessions/${sessionId}`)
  },

  respond(sessionId: string, payload: RespondRequest) {
    return requestJson<RespondResponse>(`/api/v1/review/sessions/${sessionId}/respond`, {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  },

  requestHint(sessionId: string) {
    return requestJson<HintResponse>(`/api/v1/review/sessions/${sessionId}/hint`, {
      method: 'POST',
      body: JSON.stringify({}),
    })
  },

  finish(sessionId: string) {
    return requestJson<FinishResponse>(`/api/v1/review/sessions/${sessionId}/finish`, {
      method: 'POST',
      body: JSON.stringify({}),
    })
  },
}
