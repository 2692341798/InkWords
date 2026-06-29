import { createRef } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { GeneratorInput } from './GeneratorInput'
import { GeneratorModules } from './GeneratorModules'
import { GeneratorOutline } from './GeneratorOutline'
import { GeneratorSourceStage } from './GeneratorSourceStage'
import { GeneratorConfigureStage } from './GeneratorConfigureStage'
import { GeneratorOutlineStage } from './GeneratorOutlineStage'
import { GeneratorStatus } from './GeneratorStatus'
import { Generator } from '@/pages/Generator'

const mockStreamState = {
  sourceType: 'git' as 'git' | 'file' | null,
  isScanning: false,
  isAnalyzing: false,
  isGenerating: false,
  analysisStep: -1,
  analysisMessage: '',
  analysisHistory: [] as { id: number; message: string; status?: string }[],
  progress: '',
  content: '',
  currentChapterTitle: '',
  sourceContent: '',
  scenarioMode: 'open_book_exam_review' as 'open_book_exam_review' | 'ebook_interpretation',
  gitUrl: '',
  modules: null as { path: string; name: string; description: string }[] | null,
  selectedModules: [] as string[],
  outline: null as { sort: number; title: string; summary: string }[] | null,
  chapterStatus: {} as Record<number, 'pending' | 'generating' | 'completed' | 'error'>,
  chapterPhases: {} as Record<number, string>,
  chapterContents: {} as Record<number, string>,
  chapterErrors: {} as Record<number, string>,
  chapterUsage: {} as Record<number, {
    prompt_tokens: number
    completion_tokens: number
    prompt_cache_hit_tokens: number
    prompt_cache_miss_tokens: number
  }>,
  setGitUrl: vi.fn(),
  setScenarioMode: vi.fn(),
  setModules: vi.fn(),
  setSelectedModules: vi.fn(),
  setParentBlogId: vi.fn(),
  setSourceContent: vi.fn(),
  reset: vi.fn(),
  setOutline: vi.fn(),
}

vi.mock('@/store/streamStore', () => ({
  useStreamStore: () => mockStreamState,
}))

vi.mock('@/hooks/useBlogStream', () => ({
  useBlogStream: () => ({
    scanGit: vi.fn(),
    analyzeGit: vi.fn(),
    parseFile: vi.fn(),
    analyzeParsedFile: vi.fn(),
    generateSeries: vi.fn(),
    generateSingle: vi.fn(),
    stopAnalyzing: vi.fn(),
    stopGenerating: vi.fn(),
  }),
}))

describe('Generator stage views', () => {
  beforeEach(() => {
    mockStreamState.isScanning = false
    mockStreamState.isAnalyzing = false
    mockStreamState.isGenerating = false
    mockStreamState.analysisStep = -1
    mockStreamState.analysisMessage = ''
    mockStreamState.analysisHistory = []
    mockStreamState.progress = ''
    mockStreamState.content = ''
    mockStreamState.currentChapterTitle = ''
    mockStreamState.sourceContent = ''
    mockStreamState.scenarioMode = 'open_book_exam_review'
    mockStreamState.gitUrl = ''
    mockStreamState.modules = null
    mockStreamState.selectedModules = []
    mockStreamState.outline = null
    mockStreamState.chapterStatus = {}
    mockStreamState.chapterPhases = {}
    mockStreamState.chapterContents = {}
    mockStreamState.chapterErrors = {}
    mockStreamState.chapterUsage = {}
    mockStreamState.setGitUrl.mockReset()
    mockStreamState.setScenarioMode.mockReset()
    mockStreamState.setModules.mockReset()
    mockStreamState.setSelectedModules.mockReset()
    mockStreamState.setParentBlogId.mockReset()
    mockStreamState.setSourceContent.mockReset()
    mockStreamState.reset.mockReset()
    mockStreamState.setOutline.mockReset()
  })

  afterEach(() => {
    mockStreamState.modules = null
    mockStreamState.selectedModules = []
    mockStreamState.outline = null
    mockStreamState.chapterStatus = {}
    mockStreamState.chapterPhases = {}
    mockStreamState.chapterContents = {}
    mockStreamState.chapterErrors = {}
    mockStreamState.chapterUsage = {}
  })

  it('renders a dedicated source stage wrapper around the source input choices', () => {
    const html = renderToStaticMarkup(
      <GeneratorSourceStage
        gitUrl=""
        setGitUrl={() => {}}
        isDragging={false}
        handleScan={() => {}}
        handleDragOver={() => {}}
        handleDragLeave={() => {}}
        handleDrop={() => {}}
        handleFileChange={() => {}}
        fileInputRef={createRef<HTMLInputElement>()}
        stopAnalyzing={() => {}}
      />,
    )

    expect(html).toContain('先选择资料来源')
    expect(html).toContain('确认你要解析的是 GitHub 仓库还是本地文档，完成后再进入配置解析策略')
    expect(html).toContain('解析开源项目')
    expect(html).toContain('解析本地文档')
  })

  it('renders configure and outline wrappers that own stage-level framing', () => {
    const configureHtml = renderToStaticMarkup(
      <GeneratorConfigureStage
        sourceLabel="GitHub 仓库"
        scenarioSelector={<div>场景选择器</div>}
        modulePicker={<div>模块选择器</div>}
        onBack={() => {}}
      />,
    )

    expect(configureHtml).toContain('当前来源：GitHub 仓库')
    expect(configureHtml).toContain('配置解析方式')
    expect(configureHtml).toContain('返回上一步')
    expect(configureHtml).toContain('场景选择器')
    expect(configureHtml).toContain('模块选择器')

    const outlineStageHtml = renderToStaticMarkup(
      <GeneratorOutlineStage
        lockedScenarioLabel="开卷复习"
        lockedPromptProfileLabel="心理学经典解读"
        outlineEditor={<div>大纲编辑器</div>}
        onBack={() => {}}
      />,
    )

    expect(outlineStageHtml).toContain('确认并调整大纲')
    expect(outlineStageHtml).toContain('创作场景：开卷复习')
    expect(outlineStageHtml).toContain('提示词类型：心理学经典解读')
    expect(outlineStageHtml).toContain('大纲编辑器')
    expect(outlineStageHtml).toContain('返回上一步')
  })

  it('shows configure content and inline progress together while analyzing', () => {
    const html = renderToStaticMarkup(
      <GeneratorConfigureStage
        sourceLabel="本地文档"
        scenarioSelector={<div>场景选择器</div>}
        fileSummary={<div>文件摘要</div>}
        progressPanel={<div>解析进度面板</div>}
        onBack={() => {}}
      />,
    )

    expect(html).toContain('配置解析方式')
    expect(html).toContain('场景选择器')
    expect(html).toContain('文件摘要')
    expect(html).toContain('解析进度面板')
  })

  it('shows outline editor and generation progress in the same stage', () => {
    const html = renderToStaticMarkup(
      <GeneratorOutlineStage
        lockedScenarioLabel="电子书解读"
        outlineEditor={<div>大纲编辑器</div>}
        progressPanel={<div>生成进度面板</div>}
        onBack={() => {}}
      />,
    )

    expect(html).toContain('确认并调整大纲')
    expect(html).toContain('大纲编辑器')
    expect(html).toContain('生成进度面板')
  })

  it('renders GeneratorStatus inside the outline stage when outline hosts progress', () => {
    mockStreamState.sourceType = 'git'
    mockStreamState.modules = [{ path: 'cmd', name: 'cmd', description: '入口目录' }]
    mockStreamState.outline = [{ sort: 1, title: '第一篇', summary: '摘要' }]
    mockStreamState.scenarioMode = 'ebook_interpretation'
    mockStreamState.isGenerating = true
    mockStreamState.progress = '正在生成第一篇'
    mockStreamState.chapterStatus = { 1: 'generating' }

    const html = renderToStaticMarkup(<Generator />)

    expect(html).toContain('当前步骤：确认大纲')
    expect(html).toContain('确认并调整大纲')
    expect(html).toContain('生成进度')
    expect(html).toContain('第一篇')
    expect(html).not.toContain('处理进度')
  })

  it('keeps child components free of stage-owned framing', () => {
    mockStreamState.modules = [{ path: 'cmd', name: 'cmd', description: '入口目录' }]
    mockStreamState.selectedModules = ['cmd']
    mockStreamState.outline = [{ sort: 1, title: '章节一', summary: '摘要' }]

    const inputHtml = renderToStaticMarkup(
      <GeneratorInput
        gitUrl=""
        setGitUrl={() => {}}
        isDragging={false}
        handleScan={() => {}}
        handleDragOver={() => {}}
        handleDragLeave={() => {}}
        handleDrop={() => {}}
        handleFileChange={() => {}}
        fileInputRef={createRef<HTMLInputElement>()}
        stopAnalyzing={() => {}}
      />,
    )
    expect(inputHtml).not.toContain('mb-12')
    expect(inputHtml).toContain('grid grid-cols-1 gap-4 md:grid-cols-2')

    const modulesHtml = renderToStaticMarkup(
      <GeneratorModules toggleModuleSelection={() => {}} handleAnalyze={() => {}} />,
    )
    expect(modulesHtml).not.toContain('mb-12')
    expect(modulesHtml).toContain('选择深入解析目录')

    const outlineHtml = renderToStaticMarkup(
      <GeneratorOutline
        isOutlineExpanded
        setIsOutlineExpanded={() => {}}
        setShowChapterDeleteConfirm={() => {}}
        handleGenerate={() => {}}
        stopGenerating={() => {}}
        lockedScenarioLabel="开卷复习"
      />,
    )
    expect(outlineHtml).not.toContain('当前创作场景：开卷复习')
    expect(outlineHtml).toContain('博客大纲 (1 篇)')
  })

  it('renders generator status as embeddable content instead of a full-screen overlay', () => {
    mockStreamState.isAnalyzing = true
    mockStreamState.analysisStep = 1
    mockStreamState.analysisMessage = '正在生成大纲...'
    mockStreamState.analysisHistory = [{ id: 1, message: '正在生成大纲...', status: 'outline' }]
    mockStreamState.sourceType = 'file'

    const html = renderToStaticMarkup(<GeneratorStatus />)

    expect(html).toContain('解析进度')
    expect(html).not.toContain('fixed inset-0')
    expect(html).toContain('overflow-hidden rounded-xl border border-border bg-card')
  })

  it('shows chapter error reasons in the inline generation progress panel', () => {
    mockStreamState.outline = [{ sort: 1, title: '第一篇', summary: '摘要' }]
    mockStreamState.isGenerating = true
    mockStreamState.chapterStatus = { 1: 'error' }
    mockStreamState.chapterErrors = { 1: 'DeepSeek 请求超时，请稍后重试' }

    const html = renderToStaticMarkup(<GeneratorStatus />)

    expect(html).toContain('第一篇')
    expect(html).toContain('失败原因')
    expect(html).toContain('DeepSeek 请求超时，请稍后重试')
  })

  it('shows chapter usage and cache hit rate in the inline generation progress panel', () => {
    mockStreamState.outline = [{ sort: 1, title: '第一篇', summary: '摘要' }]
    mockStreamState.isGenerating = true
    mockStreamState.chapterStatus = { 1: 'completed' }
    mockStreamState.chapterPhases = { 1: 'completed' }
    mockStreamState.chapterUsage = {
      1: {
        prompt_tokens: 1200,
        completion_tokens: 500,
        prompt_cache_hit_tokens: 900,
        prompt_cache_miss_tokens: 300,
      },
    }

    const html = renderToStaticMarkup(<GeneratorStatus />)

    expect(html).toContain('Prompt')
    expect(html).toContain('Completion')
    expect(html).toContain('缓存命中')
    expect(html).toContain('75%')
  })

  it('keeps only the inline status panel for progress feedback', () => {
    mockStreamState.isAnalyzing = true
    mockStreamState.analysisMessage = '正在分析仓库结构'
    mockStreamState.analysisStep = 1
    mockStreamState.analysisHistory = [{ id: 1, message: '已完成仓库克隆', status: 'done' }]

    const html = renderToStaticMarkup(<GeneratorStatus />)

    expect(html).toContain('解析进度')
    expect(html).toContain('已完成仓库克隆')
    expect(html).toContain('overflow-hidden rounded-xl border border-border bg-card')
  })
})
