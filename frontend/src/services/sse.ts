import { EventStreamContentType, fetchEventSource } from '@microsoft/fetch-event-source'
import { authTokenStore } from '@/lib/authTokenStore'
import { assertApiResponse } from './apiClient'

export type SSEOptions = Omit<Parameters<typeof fetchEventSource>[1], 'headers'> & {
  headers?: Record<string, string>
  requireAuth?: boolean
}

export const buildAuthHeader = (token: string | null) => {
  if (!token) return {}
  return { Authorization: `Bearer ${token}` }
}

export const fetchEventSourceWithAuth = (url: string, options: SSEOptions) => {
  const {
    headers: inputHeaders,
    requireAuth = true,
    onopen,
    onerror,
    ...requestOptions
  } = options
  const token = authTokenStore.getSnapshot()
  const headers: Record<string, string> = { ...(inputHeaders ?? {}) }
  if (requireAuth && token) {
    headers.Authorization = `Bearer ${token}`
  }

  return fetchEventSource(url, {
    ...requestOptions,
    headers,
    async onopen(response) {
      await assertApiResponse(response, '流式请求失败')
      const contentType = response.headers.get('content-type')
      if (!contentType?.startsWith(EventStreamContentType)) {
        throw new Error(`流式响应格式错误：${contentType || '缺少 Content-Type'}`)
      }
      await onopen?.(response)
    },
    onerror(error) {
      onerror?.(error)
      // Why: task mutations and streams are not safe to replay implicitly.
      // Callers can expose an explicit retry action with a new idempotency key.
      throw error
    },
  })
}
