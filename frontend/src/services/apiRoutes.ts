const pathSegment = (value: string) => encodeURIComponent(value)

/**
 * Browser code only uses same-origin paths. Service ownership is recorded here
 * so domain clients do not need to know container names or internal ports.
 */
export const apiRoutes = {
  coreApi: {
    auth: {
      captcha: '/api/v1/auth/captcha',
      login: '/api/v1/auth/login',
      register: '/api/v1/auth/register',
      bindGithub: '/api/v1/auth/bind-github',
      oauth: (provider: string) => `/api/v1/auth/oauth/${pathSegment(provider)}`,
    },
    user: {
      profile: '/api/v1/user/profile',
      stats: '/api/v1/user/stats',
      avatar: '/api/v1/user/avatar',
    },
    blogs: {
      collection: '/api/v1/blogs',
      draft: '/api/v1/blogs/draft',
      byId: (blogId: string) => `/api/v1/blogs/${pathSegment(blogId)}`,
    },
    tasks: {
      generation: '/api/v1/tasks/generation',
      parse: '/api/v1/tasks/parse',
      export: '/api/v1/tasks/export',
      byId: (taskId: string) => `/api/v1/tasks/${pathSegment(taskId)}`,
      cancel: (taskId: string) => `/api/v1/tasks/${pathSegment(taskId)}/cancel`,
      stream: (taskId: string) => `/api/v1/tasks/${pathSegment(taskId)}/stream`,
      download: (taskId: string) => `/api/v1/tasks/${pathSegment(taskId)}/download`,
    },
  },
  llmStream: {
    scan: '/api/v1/stream/scan',
    analyze: '/api/v1/stream/analyze',
    generate: '/api/v1/stream/generate',
    continueBlog: (blogId: string) => `/api/v1/blogs/${pathSegment(blogId)}/continue`,
    polishBlog: (blogId: string) => `/api/v1/blogs/${pathSegment(blogId)}/polish`,
  },
  parserService: {
    parseProject: '/api/v1/project/parse',
  },
  exportService: {
    seriesZip: (blogId: string) => `/api/v1/blogs/${pathSegment(blogId)}/export`,
    seriesPdf: (blogId: string) => `/api/v1/blogs/${pathSegment(blogId)}/export/pdf`,
    blogToObsidian: (blogId: string) => `/api/v1/blogs/${pathSegment(blogId)}/export/obsidian`,
    seriesToObsidian: (blogId: string) => `/api/v1/blogs/${pathSegment(blogId)}/export/obsidian/series`,
  },
  reviewService: {
    today: '/api/v1/review/today',
    pick: '/api/v1/review/pick',
    notes: '/api/v1/review/notes',
    history: '/api/v1/review/history',
    sessions: '/api/v1/review/sessions',
    session: (sessionId: string) => `/api/v1/review/sessions/${pathSegment(sessionId)}`,
    respond: (sessionId: string) => `/api/v1/review/sessions/${pathSegment(sessionId)}/respond`,
    hint: (sessionId: string) => `/api/v1/review/sessions/${pathSegment(sessionId)}/hint`,
    finish: (sessionId: string) => `/api/v1/review/sessions/${pathSegment(sessionId)}/finish`,
  },
} as const
