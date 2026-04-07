import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Mail, Lock, User, Loader2, ArrowRight } from 'lucide-react'

// Custom GitHub SVG icon since it was removed from lucide-react
const GithubIcon = ({ className }: { className?: string }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    className={className}
  >
    <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" />
    <path d="M9 18c-4.51 2-5-2-7-2" />
  </svg>
)

type AuthMode = 'login' | 'register' | 'forgot_password'

export function Login() {
  const [mode, setMode] = useState<AuthMode>('login')
  const [captcha, setCaptcha] = useState({ id: '', image: '', value: '' })
  const [countdown, setCountdown] = useState(0)
  
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    password: '',
    emailCode: '',
  })

  const fetchCaptcha = async () => {
    try {
      const res = await fetch('/api/v1/auth/captcha')
      const data = await res.json()
      if (data.code === 200) {
        setCaptcha(prev => ({ ...prev, id: data.data.captcha_id, image: data.data.image, value: '' }))
      }
    } catch (err: unknown) {
      console.error('获取验证码失败', err)
    }
  }

  useEffect(() => {
    if (mode === 'register' || mode === 'forgot_password') {
      fetchCaptcha()
    }
  }, [mode])

  useEffect(() => {
    let timer: NodeJS.Timeout
    if (countdown > 0) {
      timer = setTimeout(() => setCountdown(c => c - 1), 1000)
    }
    return () => clearTimeout(timer)
  }, [countdown])

  const handleSendCode = async () => {
    if (!formData.email) return setError('请输入邮箱')
    if (!captcha.value) return setError('请输入图形验证码')
    
    try {
      const res = await fetch('/api/v1/auth/send-code', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: formData.email,
          type: mode === 'register' ? 'register' : 'reset_password',
          captcha_id: captcha.id,
          captcha_value: captcha.value
        })
      })
      const data = await res.json()
      if (data.code === 200) {
        setCountdown(60)
        setError('')
      } else {
        setError(data.message)
        fetchCaptcha()
      }
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message || '发送失败')
      } else {
        setError('发送失败')
      }
    }
  }

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const errorMsg = params.get('error')
    if (errorMsg) {
      setError(decodeURIComponent(errorMsg))
      // 清除 URL 中的 error 参数，防止刷新后再次提示
      window.history.replaceState({}, document.title, window.location.pathname)
    }
  }, [])

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setFormData((prev) => ({ ...prev, [name]: value }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    const endpoint = mode === 'login' ? '/api/v1/auth/login' : mode === 'register' ? '/api/v1/auth/register' : '/api/v1/auth/reset-password'
    const payload = mode === 'login'
      ? { email: formData.email, password: formData.password }
      : mode === 'register'
      ? { username: formData.name, email: formData.email, password: formData.password, code: formData.emailCode }
      : { email: formData.email, new_password: formData.password, code: formData.emailCode }

    try {
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      })

      const data = await response.json()

      if (!response.ok || data.code !== 200) {
        throw new Error(data.message || '操作失败，请重试')
      }

      if (data.data?.token) {
        localStorage.setItem('token', data.data.token)
      }

      window.location.reload()
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message || '网络错误，请稍后重试')
      } else {
        setError('网络错误，请稍后重试')
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handleGithubLogin = () => {
    // 强制跳转到 API 而不经过前端 React Router 拦截，以便 Nginx 可以将 /api/ 正确代理到 backend:8080
    window.location.href = '/api/v1/auth/oauth/github'
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-zinc-50 p-4">
      <div className="max-w-md w-full p-8 bg-white rounded-2xl shadow-sm border border-zinc-100">
        <div className="mb-8 flex flex-col items-center">
          <div className="w-16 h-16 bg-zinc-900 text-white rounded-xl flex items-center justify-center text-2xl font-bold shadow-md mb-4">
            墨
          </div>
          <h1 className="text-2xl font-semibold mb-2 text-zinc-900">
            {mode === 'login' ? '欢迎回来' : mode === 'register' ? '创建账号' : '重置密码'}
          </h1>
          <p className="text-zinc-500 text-sm">
            {mode === 'login' ? '登录以继续使用墨言博客助手' : mode === 'register' ? '注册并随时随地开启智能写作' : '输入您的邮箱获取验证码以重置密码'}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4 mb-6">
          {mode === 'register' && (
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-zinc-700">昵称</label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-zinc-400">
                  <User className="h-4 w-4" />
                </div>
                <input
                  type="text"
                  name="name"
                  value={formData.name}
                  onChange={handleInputChange}
                  required={mode === 'register'}
                  className="w-full pl-10 pr-3 py-2 border border-zinc-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900/20 focus:border-zinc-900 transition-colors"
                  placeholder="请输入昵称"
                />
              </div>
            </div>
          )}

          <div className="space-y-1.5">
            <label className="text-sm font-medium text-zinc-700">邮箱</label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-zinc-400">
                <Mail className="h-4 w-4" />
              </div>
              <input
                type="email"
                name="email"
                value={formData.email}
                onChange={handleInputChange}
                required
                className="w-full pl-10 pr-3 py-2 border border-zinc-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900/20 focus:border-zinc-900 transition-colors"
                placeholder="name@example.com"
              />
            </div>
          </div>

          {mode !== 'login' && (
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-zinc-700">图形验证码</label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={captcha.value}
                  onChange={(e) => setCaptcha(prev => ({ ...prev, value: e.target.value }))}
                  required
                  className="flex-1 px-3 py-2 border border-zinc-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900/20 focus:border-zinc-900 transition-colors"
                  placeholder="请输入图形验证码"
                />
                <div 
                  className="w-[120px] h-[38px] border border-zinc-200 rounded-lg overflow-hidden cursor-pointer flex-shrink-0"
                  onClick={fetchCaptcha}
                  title="点击刷新验证码"
                >
                  {captcha.image ? (
                    <img src={captcha.image} alt="captcha" className="w-full h-full object-cover" />
                  ) : (
                    <div className="w-full h-full bg-zinc-100 flex items-center justify-center text-xs text-zinc-400">加载中...</div>
                  )}
                </div>
              </div>
            </div>
          )}

          {mode !== 'login' && (
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-zinc-700">邮箱验证码</label>
              <div className="flex gap-2">
                <input
                  type="text"
                  name="emailCode"
                  value={formData.emailCode}
                  onChange={handleInputChange}
                  required
                  className="flex-1 px-3 py-2 border border-zinc-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900/20 focus:border-zinc-900 transition-colors"
                  placeholder="6位验证码"
                />
                <Button
                  type="button"
                  onClick={handleSendCode}
                  disabled={countdown > 0}
                  variant="outline"
                  className="w-[120px] h-[38px] flex-shrink-0 text-sm border-zinc-200"
                >
                  {countdown > 0 ? `${countdown}s 后重试` : '获取验证码'}
                </Button>
              </div>
            </div>
          )}

          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium text-zinc-700">
                {mode === 'forgot_password' ? '新密码' : '密码'}
              </label>
              {mode === 'login' && (
                <button
                  type="button"
                  onClick={() => {
                    setMode('forgot_password')
                    setError('')
                  }}
                  className="text-xs text-zinc-500 hover:text-zinc-900 focus:outline-none"
                >
                  忘记密码？
                </button>
              )}
            </div>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-zinc-400">
                <Lock className="h-4 w-4" />
              </div>
              <input
                type="password"
                name="password"
                value={formData.password}
                onChange={handleInputChange}
                required
                className="w-full pl-10 pr-3 py-2 border border-zinc-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900/20 focus:border-zinc-900 transition-colors"
                placeholder="••••••••"
              />
            </div>
          </div>

          {error && (
            <div className="text-red-500 text-sm mt-2 bg-red-50 p-2 rounded-md">
              {error}
            </div>
          )}

          <Button
            type="submit"
            disabled={isLoading}
            className="w-full h-11 text-base mt-2 bg-zinc-900 hover:bg-zinc-800 text-white"
          >
            {isLoading ? (
              <Loader2 className="w-5 h-5 animate-spin" />
            ) : (
              <>
                {mode === 'login' ? '登录' : mode === 'register' ? '注册' : '重置密码'}
                <ArrowRight className="w-4 h-4 ml-2" />
              </>
            )}
          </Button>
        </form>

        <div className="relative mb-6">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-zinc-200"></div>
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="px-2 bg-white text-zinc-500">或通过以下方式</span>
          </div>
        </div>

        <Button
          type="button"
          variant="outline"
          onClick={handleGithubLogin}
          className="w-full h-11 text-base font-medium border-zinc-200 hover:bg-zinc-50"
        >
          <GithubIcon className="w-5 h-5 mr-2" />
          使用 GitHub {mode === 'login' ? '登录' : '注册'}
        </Button>

        <div className="mt-8 text-center text-sm text-zinc-500">
          {mode === 'login' ? '还没有账号？' : '已有账号？'}{' '}
          <button
            type="button"
            onClick={() => {
              setMode(mode === 'login' ? 'register' : 'login')
              setError('')
            }}
            className="text-zinc-900 font-medium hover:underline focus:outline-none"
          >
            {mode === 'login' ? '立即注册' : '返回登录'}
          </button>
        </div>
      </div>
    </div>
  )
}
