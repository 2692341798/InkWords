import { useEffect, useRef, useState } from 'react'
import type { ChangeEvent, DragEvent } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStream } from '@/hooks/useBlogStream'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { GeneratorConfigureStage } from '@/components/generator/GeneratorConfigureStage'
import { GeneratorOutline } from '@/components/generator/GeneratorOutline'
import { GeneratorOutlineStage } from '@/components/generator/GeneratorOutlineStage'
import { GeneratorStatus } from '@/components/generator/GeneratorStatus'
import { GeneratorSourceStage } from '@/components/generator/GeneratorSourceStage'
import { GeneratorModules } from '@/components/generator/GeneratorModules'
import { Button } from '@/components/ui/button'
import { StepStrip, type StepStripItem } from '@/components/shared/StepStrip'
import { scenarioModeOptions } from '@/lib/scenarioMode'
import { cn } from '@/lib/utils'
import { getGeneratorViewState } from './generatorViewState'

/**
 * Coordinates the step-focused generator workspace, including source input,
 * scenario locking, outline confirmation, and processing-state handoff to the
 * shared stream store and generator hooks.
 */
export function Generator() {
  const store = useStreamStore()
  const { scanGit, analyzeGit, parseFile, analyzeParsedFile, generateSeries, generateSingle, stopAnalyzing, stopGenerating } = useBlogStream()
  const viewState = getGeneratorViewState({
    sourceType: store.sourceType,
    sourceContent: store.sourceContent,
    modules: store.modules,
    outline: store.outline,
    scenarioMode: store.scenarioMode,
    resolvedPromptProfile: store.resolvedPromptProfile,
    isScanning: store.isScanning,
    isAnalyzing: store.isAnalyzing,
    isGenerating: store.isGenerating,
  })
  const gitUrl = store.gitUrl
  const setGitUrl = store.setGitUrl
  const [isDragging, setIsDragging] = useState(false)
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



  const handleScan = async () => {
    if (!gitUrl) return
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

  const handleAnalyzeFile = async () => {
    if (!store.sourceContent.trim()) return
    try {
      await analyzeParsedFile()
    } catch {
      // keep current configure stage so user can retry after adjusting the scenario
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
      if (file.size > 888 * 1024 * 1024) {
        alert('文件大小不能超过 888MB')
        return
      }
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
      if (file.size > 888 * 1024 * 1024) {
        alert('文件大小不能超过 888MB')
        if (fileInputRef.current) {
          fileInputRef.current.value = ''
        }
        return
      }
      try {
        await parseFile(file)
      } catch {
        if (fileInputRef.current) {
          fileInputRef.current.value = ''
        }
      }
    }
  }

  const renderScenarioSelector = () => (
    <div className="rounded-2xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="flex flex-col gap-2">
        <h2 className="text-lg font-medium text-zinc-900 dark:text-zinc-100">创作场景</h2>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          为什么先确定场景：这样系统会围绕同一个目标组织解析与写作，避免后面再切换导致语义漂移。
        </p>
      </div>
      <div className="mt-4 grid gap-3 md:grid-cols-3">
        {scenarioModeOptions.map((option) => (
          <button
            key={option.value}
            type="button"
            onClick={() => store.setScenarioMode(option.value)}
            className={cn(
              'rounded-xl border px-4 py-4 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-400',
              store.scenarioMode === option.value
                ? 'border-zinc-900 bg-zinc-50 dark:border-zinc-100 dark:bg-zinc-800/80'
                : 'border-zinc-200 bg-white hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:hover:border-zinc-500',
            )}
            aria-pressed={store.scenarioMode === option.value}
          >
            <div className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">{option.label}</div>
            <div className="mt-2 text-xs leading-5 text-zinc-500 dark:text-zinc-400">
              {option.description}
            </div>
          </button>
        ))}
      </div>
    </div>
  )

  const generatorSteps: StepStripItem[] = [
    { key: 'source', title: '选择来源', description: '先决定资料入口。' },
    { key: 'configure', title: '配置解析', description: '选择场景并完成解析。' },
    { key: 'outline', title: '确认大纲', description: '调整大纲并查看写作进度。' },
  ]

  const currentStepMeta = generatorSteps[viewState.currentStepIndex]

  const backFromConfigure = () => {
    // Why: 文件来源在现有 stage helper 中只要保留 sourceType=file 就会停留在配置页，
    // 因此这里需要回到完整初始态，确保“返回上一步”真的回到来源选择页。
    if (store.sourceType === 'file') {
      store.reset()
      return
    }

    store.setModules(null)
    store.setSelectedModules([])
    store.setOutline(null)
    store.setParentBlogId(null)
  }

  const backFromOutline = () => {
    store.setOutline(null)
    store.setParentBlogId(null)
    if (store.sourceType !== 'git') {
      store.setSourceContent('')
    }
  }

  return (
    <div className="flex-1 h-full overflow-y-auto custom-scrollbar">
      <div className="max-w-6xl mx-auto px-4 py-12">
        <section className="mb-8 rounded-3xl border border-zinc-200 bg-white px-8 py-10 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
          <div className="space-y-4">
            <span className="inline-flex items-center rounded-full bg-indigo-50 px-3 py-1 text-xs font-medium text-indigo-700 dark:bg-indigo-500/10 dark:text-indigo-300">
              智能生成博客
            </span>
            <div className="space-y-2">
              <h1 className="text-4xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">一键将开源项目或本地文档转化为高质量技术博客</h1>
              <p className="max-w-3xl text-sm leading-6 text-zinc-500 dark:text-zinc-400">
                页面现在只展开当前步骤，让用户先判断“我现在要做什么”，再看到与当前动作直接相关的模块。
              </p>
            </div>
          </div>
        </section>

        <section className="mb-8 rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
          <div className="mb-5 flex justify-end">
            <span className="rounded-full bg-zinc-100 px-3 py-1 text-xs font-medium text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300">
              当前步骤：{currentStepMeta.title}
            </span>
          </div>
          <StepStrip
            title="当前流程"
            description={currentStepMeta.description}
            steps={generatorSteps}
            currentStepIndex={viewState.currentStepIndex}
            variant="progress"
          />
        </section>

        <div className="space-y-8">
          {viewState.currentStage === 'source' ? (
            <GeneratorSourceStage
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
          ) : null}

          {viewState.currentStage === 'configure' ? (
            <GeneratorConfigureStage
              sourceLabel={store.sourceType === 'git' ? 'GitHub 仓库' : '本地文档'}
              scenarioSelector={renderScenarioSelector()}
              modulePicker={
                store.sourceType === 'git' ? (
                  <GeneratorModules
                    toggleModuleSelection={toggleModuleSelection}
                    handleAnalyze={handleAnalyze}
                  />
                ) : undefined
              }
              fileSummary={
                store.sourceType === 'file' ? (
                  <section className="rounded-2xl border border-zinc-200 bg-zinc-50 p-5 dark:border-zinc-800 dark:bg-zinc-950/40">
                    <h3 className="text-base font-semibold text-zinc-900 dark:text-zinc-100">已选择本地文档</h3>
                    <p className="mt-2 text-sm leading-6 text-zinc-500 dark:text-zinc-400">
                      当前文档已解析完成。先选择创作场景，再开始生成大纲。
                    </p>
                    <div className="mt-4 flex justify-end">
                      <Button
                        onClick={handleAnalyzeFile}
                        disabled={!store.sourceContent.trim() || store.isAnalyzing || store.isGenerating}
                        className="min-w-[140px]"
                      >
                        生成大纲
                      </Button>
                    </div>
                  </section>
                ) : undefined
              }
              progressPanel={viewState.progressHostStage === 'configure' ? <GeneratorStatus /> : undefined}
              onBack={backFromConfigure}
            />
          ) : null}

          {viewState.currentStage === 'outline' ? (
            <GeneratorOutlineStage
              lockedScenarioLabel={viewState.lockedScenarioLabel}
              lockedPromptProfileLabel={viewState.lockedPromptProfileLabel}
              onBack={backFromOutline}
              outlineEditor={
                <GeneratorOutline
                  isOutlineExpanded={isOutlineExpanded}
                  setIsOutlineExpanded={setIsOutlineExpanded}
                  setShowChapterDeleteConfirm={setShowChapterDeleteConfirm}
                  handleGenerate={handleGenerate}
                  stopGenerating={stopGenerating}
                  lockedScenarioLabel={null}
                />
              }
              progressPanel={viewState.progressHostStage === 'outline' ? <GeneratorStatus /> : undefined}
            />
          ) : null}
        </div>

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
    </div>
  )
}
