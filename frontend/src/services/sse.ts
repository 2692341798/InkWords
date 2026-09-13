import { EventStreamContentType, fetchEventSource } from '@microsoft/fetch-event-source'
import { assertApiResponse } from './apiClient'

export type SSEOptions = Omit<Parameters<typeof fetchEventSource>[1], 'headers'> & {
  headers?: Record<string, string>
}

export const fetchEventSourceLocal = (url: string, options: SSEOptions) => {
  const {
    headers: inputHeaders,
    onopen,
    onerror,
    ...requestOptions
  } = options
  const headers = Object.fromEntries(
    Object.entries(inputHeaders ?? {}).filter(([name]) => name.toLowerCase() !== 'authorization'),
  )

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
