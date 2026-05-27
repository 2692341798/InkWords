import { useCallback } from 'react'
import { reviewService, type ReviewCardResponse, type ReviewEntryType, type ReviewTurnResponse } from '@/services/review'
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

// Why: 复习会话需要把“创建 / 继续 / 回答 / 提示 / 结束”封装成单一交互入口，页面只关心触发动作和展示状态。
export function useKnowledgeReview() {
  const {
    currentSession,
    selectedMode,
    setCurrentSession,
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
      useReviewStore.getState().loadToday(),
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
    async (card: ReviewCardResponse, entryType: ReviewEntryType) => {
      const session = await reviewService.createSession({
        note_path: card.note_path,
        mode: selectedMode,
        entry_type: entryType,
      })
      setLatestHint(null)
      setLatestStageFeedback(null)
      setFinalFeedback(null)
      setCurrentSession(session)
      persistSessionID(session.session_id)
      return session
    },
    [persistSessionID, selectedMode, setCurrentSession, setFinalFeedback, setLatestHint, setLatestStageFeedback],
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
        next_question: response.next_question,
        turn_index: response.turn_index,
        turns: nextTurns,
      })
    },
    [currentSession, loadHistory, persistSessionID, setCurrentSession, setFinalFeedback, setLatestStageFeedback],
  )

  const requestHint = useCallback(async () => {
    if (!currentSession) {
      return
    }

    const response = await reviewService.requestHint(currentSession.session_id)
    setLatestHint(response.hint_text)
    setCurrentSession({
      ...currentSession,
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
      turns: appendTurn(currentSession.turns, 'system', 'completion', response.final_feedback.summary),
    })
    persistSessionID(null)
    await loadHistory(5)
  }, [currentSession, loadHistory, persistSessionID, setCurrentSession, setFinalFeedback])

  return {
    initialize,
    startSession,
    respond,
    requestHint,
    finish,
  }
}
