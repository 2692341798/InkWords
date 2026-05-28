import { useState, useRef, useEffect } from 'react'
import type { DragEvent, ChangeEvent } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStream } from '@/hooks/useBlogStream'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'

import { GeneratorInput } from '@/components/generator/GeneratorInput'
import { GeneratorModules } from '@/components/generator/GeneratorModules'
import { GeneratorOutline } from '@/components/generator/GeneratorOutline'
import { GeneratorStatus } from '@/components/generator/GeneratorStatus'
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
  const { scanGit, analyzeGit, parseFile, generateSeries, generateSingle, stopAnalyzing, stopGenerating } = useBlogStream()
  const viewState = getGeneratorViewState({
    sourceType: store.sourceType,
    modules: store.modules,
    outline: store.outline,
    scenarioMode: store.scenarioMode,
    isScanning: store.isScanning,
    isAnalyzing: store.isAnalyzing,
    isGenerating: store.isGenerating,
  })
  const gitUrl = store.gitUrl
  const setGitUrl = store.setGitUrl
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
      if (file.size > 888 * 1024 * 1024) {
        alert('文件大小不能超过 888MB')
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
      if (file.size > 888 * 1024 * 1024) {
        alert('文件大小不能超过 888MB')
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
    { key: 'input', title: '选择来源', description: '先确定资料入口和创作目标。' },
    { key: 'configure', title: '配置解析', description: '选择要深入分析的模块范围。' },
    { key: 'outline', title: '确认大纲', description: '只保留大纲编辑与开始生成。' },
    { key: 'processing', title: '处理进度', description: '处理中时只保留进度反馈。' },
  ]

  const currentStepMeta = generatorSteps[viewState.currentStepIndex]

  const summaryLines = [
    {
      label: '当前路径',
      value: viewState.currentStep === 'input'
        ? '选择来源'
        : viewState.currentStep === 'configure'
          ? '配置解析'
          : viewState.currentStep === 'outline'
            ? '确认大纲'
            : '处理中',
    },
    {
      label: '内容来源',
      value: store.sourceType === 'git' ? 'GitHub 仓库' : store.sourceType === 'file' ? '本地文档' : '尚未选择',
    },
    {
      label: '创作场景',
      value: scenarioModeOptions.find((option) => option.value === store.scenarioMode)?.label ?? '未设置',
    },
    {
      label: '当前结果',
      value: store.outline && store.outline.length > 0 ? `已生成 ${store.outline.length} 篇大纲` : store.modules?.length ? `已扫描 ${store.modules.length} 个目录` : '等待输入资料',
    },
  ]

  const nextActionText =
    viewState.currentStep === 'input'
      ? '下一步：选择资料来源并确认创作场景，然后进入解析流程。'
      : viewState.currentStep === 'configure'
        ? '下一步：勾选要深入解析的目录，系统会基于当前场景生成可编辑大纲。'
        : viewState.currentStep === 'outline'
          ? '下一步：只保留大纲编辑与开始生成，避免用户被前序步骤打断。'
          : '当前正在处理，请聚焦进度面板，等待系统完成当前步骤。'

  const backFromConfigure = () => {
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

        <div className="grid gap-8 lg:grid-cols-[minmax(0,1.4fr)_340px]">
          <div className="space-y-6">
            {viewState.shouldShowInputStep && (
              <>
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
                {viewState.shouldShowScenarioSelector && renderScenarioSelector()}
              </>
            )}

            {viewState.shouldShowConfigureStep && (
              <>
                <div className="rounded-2xl border border-zinc-200 bg-white p-5 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
                  <div className="flex items-center justify-between gap-4">
                    <div>
                      <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">当前只保留解析配置</h2>
                      <p className="mt-1 text-sm leading-6 text-zinc-500 dark:text-zinc-400">
                        为什么要收敛：扫描目录之后，只让用户处理“目录选择 + 场景确认”，避免再看到上传入口和后续大纲编辑。
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={backFromConfigure}
                      className="rounded-xl border border-zinc-200 px-3 py-2 text-sm text-zinc-600 transition hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-700 dark:text-zinc-300 dark:hover:border-zinc-600 dark:hover:text-zinc-100"
                    >
                      返回上一步
                    </button>
                  </div>
                </div>
                {viewState.shouldShowScenarioSelector && renderScenarioSelector()}
                {analyzingType === 'git' && (
                  <GeneratorModules
                    toggleModuleSelection={toggleModuleSelection}
                    handleAnalyze={handleAnalyze}
                  />
                )}
              </>
            )}

            {viewState.shouldShowOutlineStep && (
              <>
                <div className="rounded-2xl border border-zinc-200 bg-white p-5 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
                  <div className="flex items-center justify-between gap-4">
                    <div>
                      <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">当前只保留大纲确认</h2>
                      <p className="mt-1 text-sm leading-6 text-zinc-500 dark:text-zinc-400">
                        大纲生成后，前序输入和解析配置全部收起，用户只需要专注于调整大纲并开始生成。
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={backFromOutline}
                      className="rounded-xl border border-zinc-200 px-3 py-2 text-sm text-zinc-600 transition hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-700 dark:text-zinc-300 dark:hover:border-zinc-600 dark:hover:text-zinc-100"
                    >
                      返回上一步
                    </button>
                  </div>
                </div>
                <GeneratorOutline
                  isOutlineExpanded={isOutlineExpanded}
                  setIsOutlineExpanded={setIsOutlineExpanded}
                  setShowChapterDeleteConfirm={setShowChapterDeleteConfirm}
                  handleGenerate={handleGenerate}
                  stopGenerating={stopGenerating}
                  lockedScenarioLabel={viewState.lockedScenarioLabel}
                />
              </>
            )}

            {viewState.currentStep === 'processing' && (
              <section className="rounded-2xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
                <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">正在处理当前步骤</h2>
                <p className="mt-2 text-sm leading-6 text-zinc-500 dark:text-zinc-400">
                  处理中时，页面只保留当前流程说明，详细状态会通过进度弹层持续反馈，避免用户同时被多个操作区打断。
                </p>
              </section>
            )}
          </div>

          <aside className="h-fit rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
            <div>
              <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">任务摘要</h2>
              <p className="mt-1 text-sm leading-6 text-zinc-500 dark:text-zinc-400">
                右侧持续告诉用户：已经选了什么、下一步是什么。
              </p>
            </div>
            <div className="mt-5 space-y-3">
              {summaryLines.map((line) => (
                <div key={line.label} className="flex items-start justify-between gap-4 rounded-2xl border border-zinc-200 bg-zinc-50 px-4 py-3 dark:border-zinc-800 dark:bg-zinc-950/40">
                  <span className="text-sm text-zinc-500 dark:text-zinc-400">{line.label}</span>
                  <span className="text-right text-sm font-medium text-zinc-900 dark:text-zinc-100">{line.value}</span>
                </div>
              ))}
            </div>
            <div className="mt-5 rounded-2xl border border-dashed border-zinc-200 bg-zinc-50 px-4 py-4 text-sm leading-6 text-zinc-600 dark:border-zinc-800 dark:bg-zinc-950/40 dark:text-zinc-300">
              {nextActionText}
            </div>
          </aside>
        </div>

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
    </div>
  )
}
