import { useCallback, useState } from 'react'
import JSZip from 'jszip'
import type { BlogNode } from '@/store/blogStore'

export function useBatchExportZip(params: {
  blogs: BlogNode[]
  selectedForExport: Set<string>
  onDone: () => void
}) {
  const { blogs, selectedForExport, onDone } = params
  const [isExporting, setIsExporting] = useState(false)

  const handleBatchExport = useCallback(async () => {
    if (selectedForExport.size === 0) return
    setIsExporting(true)
    try {
      const zip = new JSZip()

      const addNodeToZip = (node: BlogNode, folder: JSZip | null, index: number) => {
        if (selectedForExport.has(node.id)) {
          const title = node.title || `未命名_${index}`
          const filename = `${title}.md`

          if (node.children && node.children.length > 0) {
            const subFolder = folder ? folder.folder(title) : zip.folder(title)
            node.children.forEach((child, idx) => addNodeToZip(child, subFolder, idx))
          } else {
            const targetFolder = folder || zip
            const prefix = node.chapter_sort > 0 ? `${String(node.chapter_sort).padStart(2, '0')}-` : ''
            targetFolder.file(`${prefix}${filename}`, `# ${title}\n\n${node.content || ''}`)
          }
        } else {
          if (node.children && node.children.length > 0) {
            node.children.forEach((child, idx) => addNodeToZip(child, folder, idx))
          }
        }
      }

      blogs.forEach((blog, idx) => addNodeToZip(blog, null, idx))

      const blob = await zip.generateAsync({ type: 'blob' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'blogs_export.zip'
      document.body.appendChild(a)
      a.click()
      URL.revokeObjectURL(url)
      document.body.removeChild(a)

      onDone()
    } catch (err) {
      console.error('Failed to export batch zip:', err)
    } finally {
      setIsExporting(false)
    }
  }, [blogs, onDone, selectedForExport])

  return {
    isExporting,
    handleBatchExport,
  }
}
