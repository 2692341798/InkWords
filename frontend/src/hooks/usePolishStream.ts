import { useCallback, useRef, useState } from 'react'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { toast } from 'sonner'
import { shouldResetPolishState } from '@/lib/polishStreamStop'

class StopStreamError extends Error {}

/**
 * 编辑器“润色（预览草稿）”的流式请求 Hook。
 * - 润色结果仅在前端暂存，用户点击“应用”前不覆盖正文
 * - 取消会中止请求并清空草稿（按产品约定）
 */
export const usePolishStream = () => {
  const abortControllerRef = useRef<AbortController | null>(null)
  const [isPolishing, setIsPolishing] = useState(false)
  const [draft, setDraft] = useState('')

  const cancelAndClear = useCallback(() => {
    abortControllerRef.current?.abort()
    abortControllerRef.current = null
    setDraft('')
    setIsPolishing(false)
  }, [])

  const start = useCallback(async (blogId: string, title: string, content: string) => {
    setDraft('')
    setIsPolishing(true)

    abortControllerRef.current?.abort()
    const ctrl = new AbortController()
    abortControllerRef.current = ctrl

    try {
      const token = localStorage.getItem('token')

      await fetchEventSource(`/api/v1/blogs/${blogId}/polish`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: token ? `Bearer ${token}` : ''
        },
        signal: ctrl.signal,
        openWhenHidden: true,
        body: JSON.stringify({ title, content }),
        async onopen(response) {
          if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
            return
          }
          if (response.headers.get('content-type')?.includes('application/json')) {
            const data = await response.json()
            if (response.status === 401) {
              localStorage.removeItem('token')
              window.location.reload()
              throw new StopStreamError('登录已过期，请重新登录')
            }
            throw new StopStreamError(data.message || data.error || '请求失败')
          }
          const text = await response.text()
          throw new StopStreamError(text || `请求失败: ${response.status} ${response.statusText}`)
        },
        onmessage(msg) {
          if (msg.event === 'chunk') {
            setDraft((prev) => prev + msg.data)
            return
          }
          if (msg.event === 'done') {
            setIsPolishing(false)
            throw new StopStreamError('done')
          }
          if (msg.event === 'error') {
            setIsPolishing(false)
            throw new StopStreamError(msg.data)
          }
        },
        onclose() {
          setIsPolishing(false)
          throw new StopStreamError('closed by server')
        },
        onerror(err: unknown) {
          if (err instanceof StopStreamError) {
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
  }, [])

  return { isPolishing, draft, start, cancelAndClear }
}
