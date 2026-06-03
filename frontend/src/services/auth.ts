import { authTokenStore } from '@/lib/authTokenStore'

interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

export interface CaptchaResponse {
  captcha_id: string
  image: string
}

export interface LoginRequest {
  email: string
  password: string
  captcha_id?: string
  captcha_value?: string
  remember_me?: boolean
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
  captcha_id: string
  captcha_value: string
}

export interface BindGithubRequest {
  email: string
  password: string
  github_id: string
  username: string
  avatar_url: string
}

export interface AuthSuccessResponse {
  token?: string
}

export function buildAuthHeaders(init?: HeadersInit) {
  const headers = new Headers(init)
  const token = authTokenStore.getSnapshot()

  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  return headers
}

async function requestJson<T>(url: string, init?: RequestInit, fallbackMessage = '请求失败') {
  const response = await fetch(url, {
    ...init,
    headers: buildAuthHeaders(init?.headers),
  })

  if (response.status === 401) {
    authTokenStore.clearToken()
    throw new Error('登录已过期，请重新登录')
  }

  const payload = (await response.json().catch(() => null)) as ApiEnvelope<T> | null
  if (!response.ok || !payload || payload.code !== 200) {
    throw new Error(payload?.message || fallbackMessage)
  }

  return payload.data
}

export const authService = {
  fetchCaptcha() {
    return requestJson<CaptchaResponse>('/api/v1/auth/captcha', undefined, '获取验证码失败')
  },

  login(payload: LoginRequest) {
    return requestJson<AuthSuccessResponse>(
      '/api/v1/auth/login',
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      },
      '登录失败，请稍后重试',
    )
  },

  register(payload: RegisterRequest) {
    return requestJson<AuthSuccessResponse>(
      '/api/v1/auth/register',
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      },
      '注册失败，请稍后重试',
    )
  },

  bindGithub(payload: BindGithubRequest) {
    return requestJson<AuthSuccessResponse>(
      '/api/v1/auth/bind-github',
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      },
      '绑定失败，请稍后重试',
    )
  },
}
