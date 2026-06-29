import { useCallback, useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { shouldResetPolishState } from '@/lib/polishStreamStop'
import { createTextChunkBuffer } from '@/lib/streamFlushBuffer'
import { fetchEventSourceWithAuth } from '@/services/sse'
import {
  buildGenerationTaskRequest,
  buildPolishTaskPayload,
  cancelGenerationTask,
  createGenerationTask,
  extractTaskChunkContent,
} from '@/services/generationTasks'

class StopStreamError extends Error {}

/**
 * 编辑器"润色（预览草稿）"的流式请求 Hook。
 * - 润色结果仅在前端暂存，用户点击"应用"前不覆盖正文
 * - 取消会中止请求并清空草稿（按产品约定）
 */
export const usePolishStream = () => {
  const abortControllerRef = useRef<AbortController | null>(null)
  const taskIdRef = useRef<string | null>(null)
  const [isPolishing, setIsPolishing] = useState(false)
  const [draft, setDraft] = useState('')

  const [draftBuffer] = useState(() =>
    createTextChunkBuffer((chunk) => {
      setDraft((previous) => previous + chunk)
    }),
  )

  useEffect(() => () => draftBuffer.cancel(), [draftBuffer])

  const cancelAndClear = useCallback(() => {
    if (taskIdRef.current) {
      void cancelGenerationTask(taskIdRef.current).catch(() => {
        // Why: 用户点击取消时优先保证本地 UI 立即收敛，后台取消失败不应阻塞交互。
      })
      taskIdRef.current = null
    }
    draftBuffer.cancel()
    abortControllerRef.current?.abort()
    abortControllerRef.current = null
    setDraft('')
    setIsPolishing(false)
  }, [draftBuffer])

  const start = useCallback(async (blogId: string, title: string, content: string) => {
    setDraft('')
    setIsPolishing(true)

    abortControllerRef.current?.abort()
    const ctrl = new AbortController()
    abortControllerRef.current = ctrl
    taskIdRef.current = null

    try {
      const task = await createGenerationTask(
        buildGenerationTaskRequest('polish', buildPolishTaskPayload(blogId, title, content)),
      )
      taskIdRef.current = task.task_id

      await fetchEventSourceWithAuth(task.stream_url, {
        method: 'GET',
        signal: ctrl.signal,
        openWhenHidden: true,
        async onopen(response) {
          if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
            return
          }
          if (response.headers.get('content-type')?.includes('application/json')) {
            const data = await response.json()
            throw new StopStreamError(data.message || data.error || '请求失败')
          }
          const text = await response.text()
          throw new StopStreamError(text || `请求失败: ${response.status} ${response.statusText}`)
        },
        onmessage(msg) {
          if (msg.event === 'chunk') {
            draftBuffer.push(extractTaskChunkContent(msg.data))
            return
          }
          if (msg.event === 'done') {
            draftBuffer.flush()
            taskIdRef.current = null
            setIsPolishing(false)
            throw new StopStreamError('done')
          }
          if (msg.event === 'error') {
            if (ctrl.signal.aborted) {
              draftBuffer.cancel()
            } else {
              draftBuffer.flush()
            }
            taskIdRef.current = null
            setIsPolishing(false)
            throw new StopStreamError(msg.data)
          }
        },
        onclose() {
          if (ctrl.signal.aborted) {
            draftBuffer.cancel()
          } else {
            draftBuffer.flush()
          }
          taskIdRef.current = null
          setIsPolishing(false)
          throw new StopStreamError('closed by server')
        },
        onerror(err: unknown) {
          if (ctrl.signal.aborted) {
            draftBuffer.cancel()
          } else {
            draftBuffer.flush()
          }
          if (err instanceof StopStreamError) {
            taskIdRef.current = null
            setIsPolishing(false)
            throw err
          }
          if (err instanceof DOMException && err.name === 'AbortError') {
            throw new StopStreamError('aborted')
          }
          const maybeError = err as { name?: unknown; message?: unknown }
          const name = typeof maybeError?.name === 'string' ? maybeError.name : ''
          const message = typeof maybeError?.message === 'string' ? maybeError.message : ''
          if (
            name === 'AbortError' ||
            message.includes('AbortError') ||
            message.includes('aborted') ||
            message.includes('Failed to fetch')
          ) {
            throw new StopStreamError('aborted')
          }
          taskIdRef.current = null
          setIsPolishing(false)
          throw err
        }
      })
    } catch (err: unknown) {
      if (err instanceof StopStreamError) {
        if (err.message === 'done' || err.message === 'aborted') {
          return
        }
        if (shouldResetPolishState(err.message)) {
          setIsPolishing(false)
        }
        if (err.message === 'closed by server') {
          toast.error('润色已中断')
          return
        }
        toast.error(err.message)
        return
      }
      const maybeError = err as { name?: unknown; message?: unknown }
      const name = typeof maybeError?.name === 'string' ? maybeError.name : ''
      const message = typeof maybeError?.message === 'string' ? maybeError.message : ''
      if (name === 'AbortError' || message.includes('AbortError') || message.includes('aborted')) return

      console.error('Failed to polish blog:', err)
      toast.error('润色失败，请稍后重试')
      setIsPolishing(false)
    }
  }, [draftBuffer])

  return { isPolishing, draft, start, cancelAndClear }
}
