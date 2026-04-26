import { useState, useRef, useEffect } from 'react'
import type { DragEvent, ChangeEvent } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStream } from '@/hooks/useBlogStream'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'

import { GeneratorInput } from './generator/GeneratorInput'
import { GeneratorModules } from './generator/GeneratorModules'
import { GeneratorOutline } from './generator/GeneratorOutline'
import { GeneratorStatus } from './generator/GeneratorStatus'

export function Generator() {
  const store = useStreamStore()
  const { scanGit, analyzeGit, parseFile, generateSeries, generateSingle, stopAnalyzing, stopGenerating } = useBlogStream()
  const [gitUrl, setGitUrl] = useState(store.gitUrl)
  const [isDragging, setIsDragging] = useState(false)
  const [analyzingType, setAnalyzingType] = useState<'git' | 'file'>('git')
  const [isOutlineExpanded, setIsOutlineExpanded] = useState(true)
  const [showChapterDeleteConfirm, setShowChapterDeleteConfirm] = useState<number | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    // Sync the expansion state with the generation process
    if (store.isGenerating) {
      setTimeout(() => setIsOutlineExpanded(false), 0)
    } else {
      setTimeout(() => setIsOutlineExpanded(true), 0)
    }
  }, [store.isGenerating])

  useEffect(() => {
    // If the global store gitUrl is reset (e.g. by "新建工作区"), sync the local input
    if (store.gitUrl === '' && gitUrl !== '') {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setGitUrl('')
    } else if (store.gitUrl !== '' && gitUrl !== store.gitUrl && store.gitUrl !== gitUrl.trim()) {
      // If we just loaded a project or store got updated from outside, sync the local input
      // Only do this if the store URL is completely different from our local URL.
      // This prevents overwriting the user's typing if they are typing a URL that matches but isn't exact.
      // Actually, if we just rely on store.gitUrl changes, we can just do it unconditionally when store.gitUrl changes.
      // But `useEffect` dependencies trigger even if the primitive value is the same? No, React bails out.
      // We'll just set it unconditionally if store.gitUrl !== '' to be safe.
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setGitUrl(store.gitUrl)
    }
  }, [store.gitUrl])

  useEffect(() => {
    // Clear modules if the user changes the git URL in the input box
    // But ONLY if we actually have modules AND the input is genuinely different from the store
    // Also, we ONLY clear if store.gitUrl is not empty (meaning a scan was successful)
    // and the user has changed the URL since then.
    if (store.gitUrl && gitUrl !== store.gitUrl && store.modules && store.modules.length > 0) {
      store.setModules(null)
      store.setSelectedModules([])
      store.setOutline(null)
      store.setParentBlogId(null)
    }
  }, [gitUrl, store.gitUrl, store.modules, store])

  const handleScan = async () => {
    if (!gitUrl) return
    setAnalyzingType('git')
    try {
      await scanGit(gitUrl)
    } catch {
      // intentionally leave gitUrl as is so user can correct their typo
    }
  }

  const handleAnalyze = async () => {
    if (!gitUrl || store.selectedModules.length === 0) return
    try {
      await analyzeGit(gitUrl, store.selectedModules)
    } catch {
      // ignore
    }
  }

  const toggleModuleSelection = (path: string) => {
    if (store.selectedModules.includes(path)) {
      store.setSelectedModules(store.selectedModules.filter(m => m !== path))
    } else {
      store.setSelectedModules([...store.selectedModules, path])
    }
  }

  const handleGenerate = () => {
    if (store.outline && store.outline.length > 0) {
      generateSeries()
    } else if (store.sourceContent) {
      generateSingle(store.sourceContent)
    }
  }

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragging(true)
  }

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragging(false)
  }

  const handleDrop = async (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragging(false)
    const file = e.dataTransfer.files[0]
    if (file) {
      if (file.size > 100 * 1024 * 1024) {
        alert('文件大小不能超过 100MB')
        return
      }
      setAnalyzingType('file')
      try {
        await parseFile(file)
      } catch {
        if (fileInputRef.current) {
          fileInputRef.current.value = ''
        }
      }
    }
  }

  const handleFileChange = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      if (file.size > 100 * 1024 * 1024) {
        alert('文件大小不能超过 100MB')
        if (fileInputRef.current) {
          fileInputRef.current.value = ''
        }
        return
      }
      setAnalyzingType('file')
      try {
        await parseFile(file)
      } catch {
        if (fileInputRef.current) {
          fileInputRef.current.value = ''
        }
      }
    }
  }

  return (
    <div className="max-w-5xl mx-auto px-4 py-12">
      <div className="text-center mb-12">
        <h1 className="text-4xl font-bold text-zinc-900 dark:text-zinc-100 mb-4 tracking-tight">智能生成博客</h1>
        <p className="text-lg text-zinc-500 dark:text-zinc-400">一键将开源项目或本地文档转化为高质量技术博客</p>
      </div>

      <GeneratorInput
        gitUrl={gitUrl}
        setGitUrl={setGitUrl}
        isDragging={isDragging}
        handleScan={handleScan}
        handleDragOver={handleDragOver}
        handleDragLeave={handleDragLeave}
        handleDrop={handleDrop}
        handleFileChange={handleFileChange}
        fileInputRef={fileInputRef}
        stopAnalyzing={stopAnalyzing}
      />

      {analyzingType === 'git' && (
        <GeneratorModules
          toggleModuleSelection={toggleModuleSelection}
          handleAnalyze={handleAnalyze}
        />
      )}

      <GeneratorOutline
        isOutlineExpanded={isOutlineExpanded}
        setIsOutlineExpanded={setIsOutlineExpanded}
        setShowChapterDeleteConfirm={setShowChapterDeleteConfirm}
        handleGenerate={handleGenerate}
        stopGenerating={stopGenerating}
      />

      <GeneratorStatus />

      <ConfirmDialog
        isOpen={showChapterDeleteConfirm !== null}
        onConfirm={() => {
          if (showChapterDeleteConfirm !== null && store.outline) {
            const newOutline = store.outline.filter((_, i) => i !== showChapterDeleteConfirm)
            store.setOutline(newOutline)
            setShowChapterDeleteConfirm(null)
          }
        }}
        onCancel={() => setShowChapterDeleteConfirm(null)}
        title="删除章节"
        message="确定要删除这个章节吗？删除后将无法恢复。"
        confirmText="删除"
        cancelText="取消"
        isDestructive={true}
      />
    </div>
  )
}
