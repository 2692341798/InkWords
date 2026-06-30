import { useEffect, useRef, useState } from 'react'
import { useDebounce } from '@/hooks/useDebounce'
import type { BlogNode } from '@/store/blogStore'

interface UseEditorAutosaveOptions {
  selectedBlog: BlogNode | null
  title: string
  content: string
  updateBlog: (id: string, data: { title?: string; content?: string }) => Promise<void>
}

/**
 * 管理编辑器的自动保存生命周期：
 * - 2 秒防抖后自动保存
 * - 组件卸载时保存未持久化的变更
 */
export function useEditorAutosave({
  selectedBlog,
  title,
  content,
  updateBlog,
}: UseEditorAutosaveOptions) {
  const [isSaving, setIsSaving] = useState(false)
  const [lastSaved, setLastSaved] = useState<Date | null>(null)

  // 跟踪最新状态，供卸载时保存使用
  const currentStateRef = useRef({ selectedBlog, title, content })
  useEffect(() => {
    currentStateRef.current = { selectedBlog, title, content }
  }, [selectedBlog, title, content])

  // 组件卸载时，若有未保存变更则立即写入
  useEffect(() => {
    return () => {
      const { selectedBlog: b, title: t, content: c } = currentStateRef.current
      if (b && (t !== b.title || c !== b.content)) {
        updateBlog(b.id, { title: t, content: c })
      }
    }
  }, [updateBlog])

  // 防抖后触发自动保存
  const debouncedTitle = useDebounce(title, 2000)
  const debouncedContent = useDebounce(content, 2000)

  useEffect(() => {
    if (selectedBlog && (debouncedTitle !== selectedBlog.title || debouncedContent !== selectedBlog.content)) {
      const save = async () => {
        setIsSaving(true)
        try {
          await updateBlog(selectedBlog.id, {
            title: debouncedTitle,
            content: debouncedContent,
          })
          setLastSaved(new Date())
        } finally {
          setIsSaving(false)
        }
      }
      save()
    }
  }, [debouncedTitle, debouncedContent, selectedBlog, updateBlog])

  return { isSaving, lastSaved }
}
