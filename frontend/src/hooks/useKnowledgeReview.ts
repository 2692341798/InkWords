import { useCallback } from 'react'
import {
  reviewService,
  type ReviewCardResponse,
  type ReviewEntryType,
  type ReviewMode,
  type ReviewTurnResponse,
} from '@/services/review'
import { useReviewStore } from '@/store/reviewStore'

const activeSessionStorageKey = 'inkwords-active-review-session'

const getStorage = () => {
  if (typeof window === 'undefined') {
    return null
  }

  return window.localStorage
}

const appendTurn = (
  turns: ReviewTurnResponse[] | undefined,
  role: string,
  turnType: string,
  content: string,
): ReviewTurnResponse[] => {
  const nextTurns = [...(turns ?? [])]
  nextTurns.push({
    turn_index: nextTurns.length + 1,
    role,
    turn_type: turnType,
    content,
  })
  return nextTurns
}

/**
 * Encapsulates the full knowledge-review session lifecycle, including restore,
 * start, answer, hint, finish, and local persistence boundaries, so the page
 * can focus on presenting the current review state.
 */
// Why: 复习会话需要把“创建 / 继续 / 回答 / 提示 / 结束”封装成单一交互入口，页面只关心触发动作和展示状态。
export function useKnowledgeReview() {
  const {
    currentSession,
    selectedMode,
    clearSessionState,
    setCurrentSession,
    setShouldResumeSessionOnOpen,
    setLatestHint,
    setLatestStageFeedback,
    setFinalFeedback,
    loadHistory,
  } = useReviewStore()

  const persistSessionID = useCallback((sessionId: string | null) => {
    const storage = getStorage()
    if (!storage) {
      return
    }

    if (sessionId) {
      storage.setItem(activeSessionStorageKey, sessionId)
      return
    }

    storage.removeItem(activeSessionStorageKey)
  }, [])

  const initialize = useCallback(async () => {
    await Promise.all([
      useReviewStore.getState().loadRecommendation(),
      useReviewStore.getState().loadHistory(5),
    ])

    const sessionId = getStorage()?.getItem(activeSessionStorageKey)
    if (!sessionId) {
      return
    }

    try {
      const restored = await reviewService.getSession(sessionId)
      setCurrentSession(restored)
    } catch {
      persistSessionID(null)
    }
  }, [persistSessionID, setCurrentSession])

  const startSession = useCallback(
    async (card: ReviewCardResponse, entryType: ReviewEntryType, modeOverride?: ReviewMode) => {
      const session = await reviewService.createSession({
        note_path: card.note_path,
        // Why: 页面可能提供“提问开始”这类显式模式入口，创建会话时必须以本次点击指定的模式为准，
        // 不能依赖尚未完成重渲染的闭包状态。
        mode: modeOverride ?? selectedMode,
        entry_type: entryType,
      })
      setLatestHint(null)
      setLatestStageFeedback(null)
      setFinalFeedback(null)
      setCurrentSession(session)
      setShouldResumeSessionOnOpen(true)
      persistSessionID(session.session_id)
      return session
    },
    [
      persistSessionID,
      selectedMode,
      setCurrentSession,
      setFinalFeedback,
      setLatestHint,
      setLatestStageFeedback,
      setShouldResumeSessionOnOpen,
    ],
  )

  const respond = useCallback(
    async (answer: string) => {
      if (!currentSession) {
        return
      }

      const response = await reviewService.respond(currentSession.session_id, { answer })
      let nextTurns = appendTurn(currentSession.turns, 'user', 'answer', answer)

      if (response.stage_feedback) {
        nextTurns = appendTurn(nextTurns, 'system', 'feedback', response.stage_feedback)
        setLatestStageFeedback(response.stage_feedback)
      } else {
        setLatestStageFeedback(null)
      }

      if (response.hint_text) {
        setLatestHint(response.hint_text)
        nextTurns = appendTurn(nextTurns, 'system', 'hint', response.hint_text)
      }

      if (response.excerpt_text) {
        nextTurns = appendTurn(nextTurns, 'system', 'excerpt', response.excerpt_text)
      }

      if (response.next_question) {
        nextTurns = appendTurn(nextTurns, 'system', 'question', response.next_question)
      }

      if (response.completed && response.final_feedback.summary) {
        nextTurns = appendTurn(nextTurns, 'system', 'completion', response.final_feedback.summary)
        setFinalFeedback(response.final_feedback)
        persistSessionID(null)
        await loadHistory(5)
      }

      setCurrentSession({
        ...currentSession,
        status: response.session_status,
        ready_to_answer: true,
        current_round_goal: response.current_round_goal,
        latest_review_feedback: response.review_feedback ?? null,
        next_question: response.next_question,
        turn_index: response.turn_index,
        turns: nextTurns,
      })
    },
    [currentSession, loadHistory, persistSessionID, setCurrentSession, setFinalFeedback, setLatestStageFeedback],
  )

  const startAnswering = useCallback(() => {
    if (!currentSession || currentSession.ready_to_answer) {
      return
    }

    setCurrentSession({
      ...currentSession,
      ready_to_answer: true,
    })
  }, [currentSession, setCurrentSession])

  const requestHint = useCallback(async () => {
    if (!currentSession) {
      return
    }

    const response = await reviewService.requestHint(currentSession.session_id)
    setLatestHint(response.hint_text)
    setCurrentSession({
      ...currentSession,
      current_round_goal: currentSession.current_round_goal,
      latest_review_feedback: currentSession.latest_review_feedback ?? null,
      turn_index: Math.max(currentSession.turn_index + 1, currentSession.turn_index),
      turns: appendTurn(currentSession.turns, 'system', 'hint', response.hint_text),
    })
  }, [currentSession, setCurrentSession, setLatestHint])

  const finish = useCallback(async () => {
    if (!currentSession) {
      return
    }

    const response = await reviewService.finish(currentSession.session_id)
    setFinalFeedback(response.final_feedback)
    setCurrentSession({
      ...currentSession,
      status: response.session_status,
      current_round_goal: currentSession.current_round_goal,
      latest_review_feedback: currentSession.latest_review_feedback ?? null,
      turns: appendTurn(currentSession.turns, 'system', 'completion', response.final_feedback.summary),
    })
    persistSessionID(null)
    await loadHistory(5)
  }, [currentSession, loadHistory, persistSessionID, setCurrentSession, setFinalFeedback])

  const clearSession = useCallback(() => {
    clearSessionState()
    persistSessionID(null)
  }, [clearSessionState, persistSessionID])

  return {
    initialize,
    startSession,
    startAnswering,
    respond,
    requestHint,
    finish,
    clearSession,
  }
}
