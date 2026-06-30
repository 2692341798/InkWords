import { requestBlob, requestEnvelope } from './apiClient'
import { apiRoutes } from './apiRoutes'

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

export const blogService = {
  fetchBlogTree() {
    return requestEnvelope<BlogServiceNode[]>(apiRoutes.coreApi.blogs.collection, {
      fallbackMessage: '获取博客列表失败',
    })
  },

  createDraftBlog() {
    return requestEnvelope<BlogServiceNode>(apiRoutes.coreApi.blogs.draft, {
      method: 'POST',
      fallbackMessage: '创建草稿失败',
    })
  },

  updateBlog(id: string, updates: { title?: string; content?: string }) {
    return requestEnvelope<null>(apiRoutes.coreApi.blogs.byId(id), {
      method: 'PUT',
      json: updates,
      fallbackMessage: '更新博客失败',
    }).then(() => undefined)
  },

  batchDeleteBlogs(ids: string[]) {
    return requestEnvelope<null>(apiRoutes.coreApi.blogs.collection, {
      method: 'DELETE',
      json: { blog_ids: ids },
      fallbackMessage: '批量删除失败',
    }).then(() => undefined)
  },

  exportBlogToObsidian(id: string) {
    return requestEnvelope<null>(apiRoutes.exportService.blogToObsidian(id), {
      method: 'POST',
      fallbackMessage: '同步到 Obsidian 失败',
    }).then(() => undefined)
  },

  exportSeriesToObsidian(id: string) {
    return requestEnvelope<null>(apiRoutes.exportService.seriesToObsidian(id), {
      method: 'POST',
      fallbackMessage: '同步系列失败',
    }).then(() => undefined)
  },

  exportSeriesZip(id: string) {
    return requestBlob(apiRoutes.exportService.seriesZip(id), {
      fallbackMessage: '导出系列失败',
    })
  },

  exportSeriesPdf(id: string) {
    return requestBlob(apiRoutes.exportService.seriesPdf(id), {
      fallbackMessage: '导出 PDF 失败',
    })
  },
}
