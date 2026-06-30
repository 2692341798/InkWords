import { requestEnvelope } from './apiClient'
import { apiRoutes } from './apiRoutes'

export { buildAuthHeaders } from './apiClient'

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

export const authService = {
  fetchCaptcha() {
    return requestEnvelope<CaptchaResponse>(apiRoutes.coreApi.auth.captcha, {
      fallbackMessage: '获取验证码失败',
    })
  },

  login(payload: LoginRequest) {
    return requestEnvelope<AuthSuccessResponse>(apiRoutes.coreApi.auth.login, {
      method: 'POST',
      json: payload,
      fallbackMessage: '登录失败，请稍后重试',
    })
  },

  register(payload: RegisterRequest) {
    return requestEnvelope<AuthSuccessResponse>(apiRoutes.coreApi.auth.register, {
      method: 'POST',
      json: payload,
      fallbackMessage: '注册失败，请稍后重试',
    })
  },

  bindGithub(payload: BindGithubRequest) {
    return requestEnvelope<AuthSuccessResponse>(apiRoutes.coreApi.auth.bindGithub, {
      method: 'POST',
      json: payload,
      fallbackMessage: '绑定失败，请稍后重试',
    })
  },
}
