import { requestEnvelope } from './apiClient'
import { apiRoutes } from './apiRoutes'

interface UserTechStackStat {
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

export const userService = {
  async getDashboardData() {
    const [stats, profile] = await Promise.all([
      requestEnvelope<UserStatsResponse>(apiRoutes.coreApi.user.stats, {
        fallbackMessage: '获取用户统计失败',
      }),
      requestEnvelope<UserProfileResponse>(apiRoutes.coreApi.user.profile, {
        fallbackMessage: '获取用户信息失败',
      }),
    ])

    return { stats, profile }
  },

  updateUsername(username: string) {
    return requestEnvelope<null>(apiRoutes.coreApi.user.profile, {
      method: 'PUT',
      json: { username },
      fallbackMessage: '更新用户名失败',
    }).then(() => undefined)
  },

  uploadAvatar(formData: FormData) {
    return requestEnvelope<{ avatar_url: string }>(apiRoutes.coreApi.user.avatar, {
      method: 'POST',
      body: formData,
      fallbackMessage: '上传头像失败',
    }).then((data) => data.avatar_url)
  },
}
