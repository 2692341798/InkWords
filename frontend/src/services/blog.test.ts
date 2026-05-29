import { beforeEach, describe, expect, it, vi } from 'vitest'
import { blogService } from './blog'

const mockFetch = vi.fn()
const storage = new Map<string, string>()

describe('blogService', () => {
  beforeEach(() => {
    mockFetch.mockReset()
    vi.stubGlobal('fetch', mockFetch)
    storage.clear()
    vi.stubGlobal('localStorage', {
      getItem: vi.fn((key: string) => storage.get(key) ?? null),
      setItem: vi.fn((key: string, value: string) => {
        storage.set(key, value)
      }),
      removeItem: vi.fn((key: string) => {
        storage.delete(key)
      }),
      clear: vi.fn(() => {
        storage.clear()
      }),
    })
    globalThis.localStorage.setItem('token', 'blog-token')
  })

  it('loads the blog tree with an auth header', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        code: 200,
        data: [{ id: 'blog-1', title: 'Guide', children: [] }],
      }),
    } as Response)

    await expect(blogService.fetchBlogTree()).resolves.toEqual([
      { id: 'blog-1', title: 'Guide', children: [] },
    ])

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/blogs')
    expect(new Headers(init.headers).get('Authorization')).toBe('Bearer blog-token')
  })

  it('creates a draft blog via POST and returns the parsed draft node', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        code: 200,
        data: {
          id: 'draft-1',
          title: '新草稿',
          content: '',
          children: [],
        },
      }),
    } as Response)

    await expect(blogService.createDraftBlog()).resolves.toMatchObject({
      id: 'draft-1',
      title: '新草稿',
    })

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/blogs/draft')
    expect(init.method).toBe('POST')
  })

  it('updates a blog with a JSON payload', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        code: 200,
        data: null,
      }),
    } as Response)

    await expect(
      blogService.updateBlog('blog-1', {
        title: 'Updated',
        content: 'Updated content',
      }),
    ).resolves.toBeUndefined()

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/blogs/blog-1')
    expect(init.method).toBe('PUT')
    expect(new Headers(init.headers).get('Content-Type')).toBe('application/json')
    expect(init.body).toBe(JSON.stringify({ title: 'Updated', content: 'Updated content' }))
  })

  it('sends a batch delete request with the selected blog ids', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        code: 200,
        data: null,
      }),
    } as Response)

    await expect(blogService.batchDeleteBlogs(['blog-1', 'blog-2'])).resolves.toBeUndefined()

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/blogs')
    expect(init.method).toBe('DELETE')
    expect(init.body).toBe(JSON.stringify({ blog_ids: ['blog-1', 'blog-2'] }))
  })

  it('returns a PDF blob for series export', async () => {
    const pdfBlob = new Blob(['pdf-content'], { type: 'application/pdf' })
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      blob: async () => pdfBlob,
    } as Response)

    await expect(blogService.exportSeriesPdf('series-1')).resolves.toBe(pdfBlob)

    const [url] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/blogs/series-1/export/pdf')
  })

  it('clears the token and throws a normalized error on unauthorized responses', async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({
        code: 401,
        message: '登录已过期，请重新登录',
      }),
    } as Response)

    await expect(blogService.fetchBlogTree()).rejects.toThrow('登录已过期，请重新登录')

    expect(globalThis.localStorage.getItem('token')).toBeNull()
  })
})
