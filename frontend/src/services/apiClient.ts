import { authTokenStore } from '@/lib/authTokenStore'

export const AUTH_EXPIRED_MESSAGE = '登录已过期，请重新登录'
export const GATEWAY_UNAVAILABLE_MESSAGE = '服务暂时不可用，请稍后重试'

interface ApiEnvelope<T> {
  code: number
  message?: string
  error?: string
  data: T
}

interface ErrorPayload {
  message?: string
  error?: string
}

export interface ApiRequestOptions extends Omit<RequestInit, 'body'> {
  body?: BodyInit | null
  json?: unknown
  fallbackMessage?: string
  fetchImpl?: typeof fetch
  token?: string | null
}

const gatewayErrorStatuses = new Set([502, 503, 504])

export function buildAuthHeaders(init?: HeadersInit, token = authTokenStore.getSnapshot()) {
  const headers = new Headers(init)
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  return headers
}

const readJson = async (response: Response) => {
  return response.json().catch(() => null) as Promise<unknown>
}

const readErrorMessage = async (response: Response, fallbackMessage: string) => {
  const payload = (await readJson(response)) as ErrorPayload | null
  return payload?.message || payload?.error || fallbackMessage
}

export async function assertApiResponse(response: Response, fallbackMessage = '请求失败') {
  if (response.status === 401) {
    authTokenStore.clearToken()
    throw new Error(AUTH_EXPIRED_MESSAGE)
  }
  if (gatewayErrorStatuses.has(response.status)) {
    throw new Error(GATEWAY_UNAVAILABLE_MESSAGE)
  }
  if (!response.ok) {
    throw new Error(await readErrorMessage(response, fallbackMessage))
  }
}

const executeRequest = async (url: string, options: ApiRequestOptions) => {
  const {
    body,
    json,
    fallbackMessage: _fallbackMessage,
    fetchImpl = fetch,
    token = authTokenStore.getSnapshot(),
    headers: inputHeaders,
    ...init
  } = options
  void _fallbackMessage
  const headers = buildAuthHeaders(inputHeaders, token)
  let requestBody = body

  if (json !== undefined) {
    headers.set('Content-Type', 'application/json')
    requestBody = JSON.stringify(json)
  }

  return fetchImpl(url, {
    ...init,
    headers,
    body: requestBody,
  })
}

export async function requestJson<T>(url: string, options: ApiRequestOptions = {}): Promise<T> {
  const fallbackMessage = options.fallbackMessage ?? '请求失败'
  const response = await executeRequest(url, options)
  await assertApiResponse(response, fallbackMessage)
  const payload = await readJson(response)
  if (payload === null) {
    throw new Error(fallbackMessage)
  }
  return payload as T
}

export async function requestEnvelope<T>(url: string, options: ApiRequestOptions = {}): Promise<T> {
  const fallbackMessage = options.fallbackMessage ?? '请求失败'
  const payload = await requestJson<ApiEnvelope<T>>(url, options)
  if (payload.code !== 200) {
    throw new Error(payload.message || payload.error || fallbackMessage)
  }
  return payload.data
}

export async function requestBlob(url: string, options: ApiRequestOptions = {}): Promise<Blob> {
  const fallbackMessage = options.fallbackMessage ?? '下载失败'
  const response = await executeRequest(url, options)
  await assertApiResponse(response, fallbackMessage)
  return response.blob()
}
