import { create } from 'zustand'

export interface BlogNode {
  id: string
  title: string
  content: string
  source_type: string
  status: number
  chapter_sort: number
  parent_id: string | null
  created_at: string
  updated_at: string
  children: BlogNode[]
}

interface BlogState {
  blogs: BlogNode[]
  isLoading: boolean
  selectedBlog: BlogNode | null
  currentView: 'generator' | 'dashboard'
  fetchBlogs: () => Promise<void>
  selectBlog: (blog: BlogNode | null) => void
  setCurrentView: (view: 'generator' | 'dashboard') => void
  updateBlog: (id: string, updates: { title?: string; content?: string }) => Promise<void>
  updateBlogLocal: (id: string, updates: { title?: string; content?: string }) => void
  batchDeleteBlogs: (ids: string[]) => Promise<void>
}

export const useBlogStore = create<BlogState>((set, get) => ({
  blogs: [],
  isLoading: false,
  selectedBlog: null,
  currentView: 'generator',
  
  fetchBlogs: async () => {
    set({ isLoading: true })
    try {
      const token = localStorage.getItem('token')
      const res = await fetch('/api/v1/blogs', {
        headers: {
          'Authorization': token ? `Bearer ${token}` : ''
        }
      })
      const text = await res.text()
      if (!text) {
        console.log('Empty response received')
        return
      }
      
      try {
        const json = JSON.parse(text)
        if (json.code === 200) {
          set({ blogs: json.data || [] })
        }
      } catch (e) {
        console.error('Failed to parse JSON:', e, text)
      }
    } catch (error) {
      console.error('Failed to fetch blogs:', error)
    } finally {
      set({ isLoading: false })
    }
  },

  selectBlog: (blog) => {
    set({ selectedBlog: blog })
  },

  setCurrentView: (view) => {
    set({ currentView: view, selectedBlog: null })
  },

  updateBlogLocal: (id, updates) => {
    set((state) => {
      // Helper function to recursively update a blog in the tree
      const updateNode = (nodes: BlogNode[]): BlogNode[] => {
        return nodes.map(node => {
          if (node.id === id) {
            return { ...node, ...updates }
          }
          if (node.children && node.children.length > 0) {
            return { ...node, children: updateNode(node.children) }
          }
          return node
        })
      }
      
      const newBlogs = updateNode(state.blogs)
      
      let newSelectedBlog = state.selectedBlog
      if (state.selectedBlog?.id === id) {
        newSelectedBlog = { ...state.selectedBlog, ...updates }
      }
      
      return { blogs: newBlogs, selectedBlog: newSelectedBlog }
    })
  },

  updateBlog: async (id, updates) => {
    try {
      // Optimistic update
      get().updateBlogLocal(id, updates)

      const token = localStorage.getItem('token')
      const res = await fetch(`/api/v1/blogs/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : ''
        },
        body: JSON.stringify(updates)
      })
      
      if (!res.ok) {
        throw new Error('Failed to update blog')
      }
    } catch (error) {
      console.error('Failed to update blog:', error)
      // We could revert the optimistic update here if needed
      // get().fetchBlogs()
    }
  },

  batchDeleteBlogs: async (ids: string[]) => {
    try {
      const token = localStorage.getItem('token')
      const res = await fetch('/api/v1/blogs', {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : ''
        },
        body: JSON.stringify({ blog_ids: ids })
      })

      if (!res.ok) {
        throw new Error('Failed to batch delete blogs')
      }

      // 刷新列表并清除选中状态
      await get().fetchBlogs()
      
      // 如果当前选中的博客被删除了，或者它的父节点被删除了，清除选中状态
      const selectedBlog = get().selectedBlog
      if (selectedBlog && (ids.includes(selectedBlog.id) || (selectedBlog.parent_id && ids.includes(selectedBlog.parent_id)))) {
        get().selectBlog(null)
      }
    } catch (error) {
      console.error('Failed to batch delete blogs:', error)
      throw error
    }
  }
}))
