import { useRef, useState } from 'react'
import { fetchEventSourceWithAuth } from '@/services/sse'
import {
  buildContinueTaskPayload,
  buildGenerationTaskRequest,
  createGenerationTask,
  extractTaskChunkContent,
} from '@/services/generationTasks'

class StopStreamError extends Error {}

interface UseContinueStreamOptions {
  blogId: string
  getContent: () => string
  setContent: (content: string) => void
  onDone: (content: string) => void
}

/**
 * 管理"继续生成"的 SSE 流式请求生命周期。
 * 封装了任务的创建、chunk 累积、错误/完成处理。
 */
export function useContinueStream({
  blogId,
  getContent,
  setContent,
  onDone,
}: UseContinueStreamOptions) {
  const [isContinuing, setIsContinuing] = useState(false)
  const isContinuingRef = useRef(false)

  const handleContinueGenerating = async () => {
    if (isContinuingRef.current) return
    isContinuingRef.current = true
    setIsContinuing(true)

    try {
      let currentContent = getContent()

      const task = await createGenerationTask(
        buildGenerationTaskRequest('continue', buildContinueTaskPayload(blogId)),
      )

      await fetchEventSourceWithAuth(task.stream_url, {
        method: 'GET',
        async onopen(response) {
          if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
            return
          }
          if (response.headers.get('content-type')?.includes('application/json')) {
            const data = await response.json()
            throw new StopStreamError(data.error || '请求失败')
          }
          const text = await response.text()
          throw new StopStreamError(text || `请求失败: ${response.status} ${response.statusText}`)
        },
        onmessage(msg) {
          if (msg.event === 'chunk') {
            currentContent += extractTaskChunkContent(msg.data)
            setContent(currentContent)
          } else if (msg.event === 'done') {
            onDone(currentContent)
            isContinuingRef.current = false
            setIsContinuing(false)
            throw new StopStreamError('done')
          } else if (msg.event === 'error') {
            console.error('Continue generating error:', msg.data)
            isContinuingRef.current = false
            setIsContinuing(false)
            throw new StopStreamError(msg.data)
          }
        },
        onclose() {
          isContinuingRef.current = false
          setIsContinuing(false)
          throw new StopStreamError('closed by server')
        },
        onerror(err: unknown) {
          if (err instanceof StopStreamError) {
            throw err
          }
          if (err instanceof DOMException && err.name === 'AbortError') {
            throw new StopStreamError('aborted')
          }
          const maybeError = err as { name?: unknown; message?: unknown }
          const name = typeof maybeError?.name === 'string' ? maybeError.name : ''
          const message = typeof maybeError?.message === 'string' ? maybeError.message : ''
          if (name === 'AbortError' || message.includes('AbortError') || message.includes('aborted') || message.includes('Failed to fetch')) {
            throw new StopStreamError('aborted')
          }
          console.error('Continue generating fetch error:', err)
          isContinuingRef.current = false
          setIsContinuing(false)
          throw err
        }
      })
    } catch (err: unknown) {
      if (err instanceof StopStreamError) {
        if (err.message === 'done' || err.message === 'aborted') {
          return
        }
      }
      const maybeError = err as { name?: unknown; message?: unknown }
      const name = typeof maybeError?.name === 'string' ? maybeError.name : ''
      const message = typeof maybeError?.message === 'string' ? maybeError.message : ''
      if (name === 'AbortError' || message.includes('AbortError') || message.includes('aborted')) return

      console.error('Failed to continue generating:', err)
      isContinuingRef.current = false
      setIsContinuing(false)
    }
  }

  return { isContinuing, handleContinueGenerating }
}
