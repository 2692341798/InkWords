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
  fetchBlogs: () => Promise<void>
  selectBlog: (blog: BlogNode | null) => void
  updateBlog: (id: string, updates: { title?: string; content?: string }) => Promise<void>
  updateBlogLocal: (id: string, updates: { title?: string; content?: string }) => void
}

export const useBlogStore = create<BlogState>((set, get) => ({
  blogs: [],
  isLoading: false,
  selectedBlog: null,
  
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
  }
}))
