import { fetchEventSource } from '@microsoft/fetch-event-source'
import { authTokenStore } from '@/lib/authTokenStore'

export type SSEOptions = Omit<Parameters<typeof fetchEventSource>[1], 'headers'> & {
  headers?: Record<string, string>
  requireAuth?: boolean
}

export const buildAuthHeader = (token: string | null) => {
  if (!token) return {}
  return { Authorization: `Bearer ${token}` }
}

export const fetchEventSourceWithAuth = (url: string, options: SSEOptions) => {
  const token = authTokenStore.getSnapshot()
  const headers: Record<string, string> = { ...(options.headers ?? {}) }
  if (options.requireAuth !== false && token) {
    headers.Authorization = `Bearer ${token}`
  }

  return fetchEventSource(url, {
    ...options,
    headers,
    async onopen(response) {
      if (response.status === 401) {
        authTokenStore.clearToken()
        throw new Error('登录已过期，请重新登录')
      }
      await options.onopen?.(response)
    },
  })
}
