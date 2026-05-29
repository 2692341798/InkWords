import { buildAuthHeaders } from './auth'

interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

export interface UserTechStackStat {
  name: string
  count: number
}

export interface UserStatsResponse {
  tokens_used: number
  estimated_cost: number
  total_articles: number
  total_words: number
  tech_stack_stats: UserTechStackStat[]
}

export interface UserProfileResponse {
  username: string
  email: string
  avatar_url: string
  subscription_tier: number
  token_limit: number
}

const getLocalStorage = () => {
  if (typeof window === 'undefined' && typeof globalThis.localStorage === 'undefined') {
    return null
  }
  return globalThis.localStorage ?? null
}

async function requestJson<T>(url: string, init?: RequestInit, fallbackMessage = '请求用户接口失败') {
  const response = await fetch(url, {
    ...init,
    headers: buildAuthHeaders(init?.headers),
  })

  if (response.status === 401) {
    getLocalStorage()?.removeItem('token')
    throw new Error('登录已过期，请重新登录')
  }

  const payload = (await response.json().catch(() => null)) as ApiEnvelope<T> | null
  if (!response.ok || !payload || payload.code !== 200) {
    throw new Error(payload?.message || fallbackMessage)
  }

  return payload.data
}

export const userService = {
  async getDashboardData() {
    const [stats, profile] = await Promise.all([
      requestJson<UserStatsResponse>('/api/v1/user/stats', undefined, '获取用户统计失败'),
      requestJson<UserProfileResponse>('/api/v1/user/profile', undefined, '获取用户信息失败'),
    ])

    return { stats, profile }
  },

  updateUsername(username: string) {
    return requestJson<null>(
      '/api/v1/user/profile',
      {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username }),
      },
      '更新用户名失败',
    ).then(() => undefined)
  },

  uploadAvatar(formData: FormData) {
    return requestJson<{ avatar_url: string }>(
      '/api/v1/user/avatar',
      {
        method: 'POST',
        body: formData,
      },
      '上传头像失败',
    ).then((data) => data.avatar_url)
  },
}
