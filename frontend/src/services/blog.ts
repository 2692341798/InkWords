import { buildAuthHeaders } from './auth'

interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

export interface BlogServiceNode {
  id: string
  title: string
  content: string
  source_type: string
  status: number
  chapter_sort: number
  parent_id: string | null
  created_at: string
  updated_at: string
  children: BlogServiceNode[]
}

const getLocalStorage = () => {
  if (typeof window === 'undefined' && typeof globalThis.localStorage === 'undefined') {
    return null
  }
  return globalThis.localStorage ?? null
}

async function requestJson<T>(url: string, init?: RequestInit, fallbackMessage = '请求博客接口失败') {
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

async function requestBlob(url: string, init?: RequestInit, fallbackMessage = '下载失败') {
  const response = await fetch(url, {
    ...init,
    headers: buildAuthHeaders(init?.headers),
  })

  if (response.status === 401) {
    getLocalStorage()?.removeItem('token')
    throw new Error('登录已过期，请重新登录')
  }

  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as ApiEnvelope<unknown> | null
    throw new Error(payload?.message || fallbackMessage)
  }

  return response.blob()
}

export const blogService = {
  fetchBlogTree() {
    return requestJson<BlogServiceNode[]>('/api/v1/blogs', undefined, '获取博客列表失败')
  },

  createDraftBlog() {
    return requestJson<BlogServiceNode>(
      '/api/v1/blogs/draft',
      { method: 'POST' },
      '创建草稿失败',
    )
  },

  updateBlog(id: string, updates: { title?: string; content?: string }) {
    return requestJson<null>(
      `/api/v1/blogs/${id}`,
      {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updates),
      },
      '更新博客失败',
    ).then(() => undefined)
  },

  batchDeleteBlogs(ids: string[]) {
    return requestJson<null>(
      '/api/v1/blogs',
      {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ blog_ids: ids }),
      },
      '批量删除失败',
    ).then(() => undefined)
  },

  exportBlogToObsidian(id: string) {
    return requestJson<null>(
      `/api/v1/blogs/${id}/export/obsidian`,
      { method: 'POST' },
      '同步到 Obsidian 失败',
    ).then(() => undefined)
  },

  exportSeriesToObsidian(id: string) {
    return requestJson<null>(
      `/api/v1/blogs/${id}/export/obsidian/series`,
      { method: 'POST' },
      '同步系列失败',
    ).then(() => undefined)
  },

  exportSeriesZip(id: string) {
    return requestBlob(`/api/v1/blogs/${id}/export`, undefined, '导出系列失败')
  },

  exportSeriesPdf(id: string) {
    return requestBlob(`/api/v1/blogs/${id}/export/pdf`, undefined, '导出 PDF 失败')
  },
}
