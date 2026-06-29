import { create } from 'zustand'
import {
  defaultScenarioModeForSource,
  type ScenarioMode,
} from '@/lib/scenarioMode'
import {
  createChapterChunkBuffer,
  createTextChunkBuffer,
} from '@/lib/streamFlushBuffer'

export interface Chapter {
  id?: string
  title: string
  summary: string
  sort: number
  files?: string[]
  action?: 'new' | 'regenerate' | 'skip' | string
}

export interface ModuleCard {
  path: string
  name: string
  description: string
}

export interface ResolvedPromptProfile {
  key: string
  displayName: string
  documentKind: string
  reason: string
}

export type ChapterPhase =
  | 'pending'
  | 'understanding'
  | 'drafting'
  | 'reviewing'
  | 'repairing'
  | 'revising'
  | 'streaming'
  | 'completed'
  | 'error'

export interface ChapterUsage {
  prompt_tokens: number
  completion_tokens: number
  prompt_cache_hit_tokens: number
  prompt_cache_miss_tokens: number
}

interface StreamState {
  sourceType: 'git' | 'file' | null
  sourceContent: string
  gitUrl: string
  scenarioMode: ScenarioMode
  modules: ModuleCard[] | null
  selectedModules: string[]
  seriesTitle: string
  outline: Chapter[] | null
  chapterStatus: Record<number, 'pending' | 'generating' | 'completed' | 'error'>
  chapterPhases: Record<number, ChapterPhase>
  chapterErrors: Record<number, string>
  chapterUsage: Record<number, ChapterUsage>
  chapterContents: Record<number, string>
  isScanning: boolean
  isAnalyzing: boolean
  isGenerating: boolean
  analysisStep: number
  analysisMessage: string
  analysisHistory: { id: number; message: string; status?: string }[]
  resolvedPromptProfile: ResolvedPromptProfile | null
  classificationStatus: 'idle' | 'classifying' | 'resolved' | 'fallback'
  classificationReason: string
  progress: string
  content: string
  currentChapterTitle: string
  abortController: AbortController | null
  parentBlogId: string | null
  currentTaskId: string | null
  setSource: (type: 'git' | 'file', content: string, gitUrl?: string) => void
  setSourceContent: (content: string) => void
  setSeriesTitle: (title: string) => void
  setOutline: (outline: Chapter[] | null) => void
  updateChapter: (sort: number, field: 'title' | 'summary', value: string) => void
  addChapter: () => void
  removeChapter: (sort: number) => void
  moveChapter: (sort: number, direction: 'up' | 'down') => void
  updateChapterStatus: (sort: number, status: 'pending' | 'generating' | 'completed' | 'error') => void
  updateChapterPhase: (sort: number, phase: ChapterPhase) => void
  setChapterUsage: (sort: number, usage: ChapterUsage) => void
  setChapterError: (sort: number, message: string) => void
  clearChapterError: (sort: number) => void
  appendChapterContent: (sort: number, content: string) => void
  bufferChapterContent: (sort: number, content: string) => void
  flushBufferedChapterContents: () => void
  clearBufferedChapterContents: () => void
  setScanning: (status: boolean) => void
  setGenerating: (status: boolean) => void
  setAnalyzing: (status: boolean) => void
  setAnalysisStep: (step: number) => void
  setAnalysisMessage: (msg: string) => void
  appendAnalysisHistory: (item: { message: string; status?: string }) => void
  clearAnalysisHistory: () => void
  setResolvedPromptProfile: (
    profile: ResolvedPromptProfile | null,
    status?: StreamState['classificationStatus'],
  ) => void
  setProgress: (msg: string) => void
  setContent: (content: string) => void
  appendContent: (chunk: string) => void
  bufferContent: (chunk: string) => void
  flushBufferedContent: () => void
  clearBufferedContent: () => void
  setCurrentChapterTitle: (title: string) => void
  setAbortController: (ctrl: AbortController | null) => void
  setParentBlogId: (id: string | null) => void
  setCurrentTaskId: (taskId: string | null) => void
  setGitUrl: (url: string) => void
  setScenarioMode: (mode: ScenarioMode) => void
  setModules: (modules: ModuleCard[] | null) => void
  setSelectedModules: (paths: string[]) => void
  reset: () => void
}

export const useStreamStore = create<StreamState>((set, get) => {
  // Why: 高频 SSE chunk 不应该每次都触发 Zustand 写入，否则会把生成期间的主线程时间浪费在无效重渲染上。
  const chapterContentBuffer = createChapterChunkBuffer((updates) => {
    set((state) => {
      const nextContents = { ...state.chapterContents }
      for (const [sort, chunk] of Object.entries(updates)) {
        const key = Number(sort)
        nextContents[key] = (nextContents[key] ?? '') + chunk
      }
      return { chapterContents: nextContents }
    })
  })
  const contentBuffer = createTextChunkBuffer((chunk) =>
    set((state) => ({ content: state.content + chunk })),
  )

  return {
    sourceType: null,
    sourceContent: '',
    gitUrl: '',
    scenarioMode: defaultScenarioModeForSource(null),
    modules: null,
    selectedModules: [],
    seriesTitle: '',
    outline: null,
    chapterStatus: {},
    chapterPhases: {},
    chapterErrors: {},
    chapterUsage: {},
    chapterContents: {},
    isScanning: false,
    isAnalyzing: false,
    isGenerating: false,
    analysisStep: -1,
    analysisMessage: '',
    analysisHistory: [],
    resolvedPromptProfile: null,
    classificationStatus: 'idle',
    classificationReason: '',
    progress: '',
    content: '',
    currentChapterTitle: '',
    abortController: null,
    parentBlogId: null,
    currentTaskId: null,
    setSource: (type, content, gitUrl) =>
      set((state) => ({
        sourceType: type,
        sourceContent: content,
        gitUrl: gitUrl || '',
        // Why: 用户手动选择的创作场景优先级高于来源推荐，上传文件或切换来源时不应把它重置掉。
        scenarioMode: state.scenarioMode,
      })),
    setSourceContent: (content) => set({ sourceContent: content }),
    setSeriesTitle: (title) => set({ seriesTitle: title }),
    setScenarioMode: (mode) => set({ scenarioMode: mode }),
    setOutline: (outline) =>
      set({
        outline,
        chapterStatus: outline
          ? outline.reduce((acc, ch) => ({ ...acc, [ch.sort]: 'pending' }), {})
          : {},
        chapterPhases: outline
          ? outline.reduce((acc, ch) => ({ ...acc, [ch.sort]: 'pending' }), {})
          : {},
        chapterErrors: {},
        chapterUsage: {},
        chapterContents: outline ? outline.reduce((acc, ch) => ({ ...acc, [ch.sort]: '' }), {}) : {},
      }),
  updateChapter: (sort, field, value) => set((state) => ({
    outline: state.outline?.map(ch => 
      ch.sort === sort ? { ...ch, [field]: value } : ch
    )
  })),
  addChapter: () => set((state) => {
    if (!state.outline) return state;
    const maxSort = state.outline.reduce((max, ch) => Math.max(max, ch.sort), 0);
    const newChapter: Chapter = {
      sort: maxSort + 1,
      title: '新章节标题',
      summary: '请填写章节摘要...',
      files: [],
      action: 'new'
    };
    return { outline: [...state.outline, newChapter] };
  }),
  removeChapter: (sort) => set((state) => {
    if (!state.outline) return state;
    const newOutline = state.outline
      .filter(ch => ch.sort !== sort)
      .map((ch, index) => ({ ...ch, sort: index + 1 }));
    const newStatus = { ...state.chapterStatus };
    const newErrors = { ...state.chapterErrors };
    const newContents = { ...state.chapterContents };
    delete newStatus[sort];
    delete newErrors[sort];
    delete newContents[sort];
    return { outline: newOutline, chapterStatus: newStatus, chapterErrors: newErrors, chapterContents: newContents };
  }),
  moveChapter: (sort, direction) => set((state) => {
    if (!state.outline) return state;
    const index = state.outline.findIndex(ch => ch.sort === sort);
    if (
      (direction === 'up' && index === 0) || 
      (direction === 'down' && index === state.outline.length - 1)
    ) return state;

    const newOutline = [...state.outline];
    const swapIndex = direction === 'up' ? index - 1 : index + 1;
    
    [newOutline[index], newOutline[swapIndex]] = [newOutline[swapIndex], newOutline[index]];
    
    const sortedOutline = newOutline.map((ch, i) => ({ ...ch, sort: i + 1 }));
    return { outline: sortedOutline };
  }),
  updateChapterStatus: (sort, status) =>
    set((state) => ({
      chapterStatus: {
        ...state.chapterStatus,
        [sort]: status
      }
    })),
  updateChapterPhase: (sort, phase) =>
    set((state) => ({
      chapterPhases: {
        ...state.chapterPhases,
        [sort]: phase,
      },
    })),
  setChapterUsage: (sort, usage) =>
    set((state) => ({
      chapterUsage: {
        ...state.chapterUsage,
        [sort]: usage,
      },
    })),
  setChapterError: (sort, message) =>
    set((state) => ({
      chapterErrors: {
        ...state.chapterErrors,
        [sort]: message,
      },
    })),
  clearChapterError: (sort) =>
    set((state) => {
      if (!(sort in state.chapterErrors)) {
        return state
      }
      const nextErrors = { ...state.chapterErrors }
      delete nextErrors[sort]
      return { chapterErrors: nextErrors }
    }),
  appendChapterContent: (sort, chunk) =>
    set((state) => ({
      chapterContents: {
        ...state.chapterContents,
        [sort]: (state.chapterContents[sort] || '') + chunk
      }
    })),
  bufferChapterContent: (sort, chunk) => {
    chapterContentBuffer.push(sort, chunk)
  },
  flushBufferedChapterContents: () => {
    chapterContentBuffer.flush()
  },
  clearBufferedChapterContents: () => {
    chapterContentBuffer.cancel()
  },
  setScanning: (status) => set({ isScanning: status }),
  setGenerating: (status) => set({ isGenerating: status }),
  setAnalyzing: (status) => set({ isAnalyzing: status }),
  setAnalysisStep: (step) => set({ analysisStep: step }),
  setAnalysisMessage: (msg) => set({ analysisMessage: msg }),
  appendAnalysisHistory: (item) => set((state) => {
    const history = [...state.analysisHistory]
    if (history.length > 0) {
      const last = history[history.length - 1]
      // If the status is the same and it's cloning/scanning/analyzing progress, update the last message
      if (last.status === item.status && (item.status === 'cloning' || item.status === 'scanning' || item.status === 'analyzing')) {
        last.message = item.message
        return { analysisHistory: history }
      }
    }
    return { analysisHistory: [...history, { id: Date.now() + Math.random(), ...item }] }
  }),
  clearAnalysisHistory: () => set({ analysisHistory: [] }),
  setResolvedPromptProfile: (profile, status = 'idle') =>
    set({
      resolvedPromptProfile: profile,
      classificationStatus: status,
      classificationReason: profile?.reason ?? '',
    }),
  setProgress: (msg) => set({ progress: msg }),
  setContent: (content) => set({ content }),
  appendContent: (chunk) => set((state) => ({ content: state.content + chunk })),
  bufferContent: (chunk) => {
    contentBuffer.push(chunk)
  },
  flushBufferedContent: () => {
    contentBuffer.flush()
  },
  clearBufferedContent: () => {
    contentBuffer.cancel()
  },
  setCurrentChapterTitle: (title) => set({ currentChapterTitle: title }),
  setAbortController: (ctrl) => set({ abortController: ctrl }),
  setParentBlogId: (id) => set({ parentBlogId: id }),
  setCurrentTaskId: (taskId) => set({ currentTaskId: taskId }),
  setGitUrl: (url) => set((state) => {
    // If the user changes the git URL in the input box and we have modules, clear them
    if (state.gitUrl && url !== state.gitUrl && state.modules && state.modules.length > 0) {
      return { 
        gitUrl: url,
        modules: null,
        selectedModules: [],
        outline: null,
        parentBlogId: null
      }
    }
    return { gitUrl: url }
  }),
  setModules: (modules) => set({ modules }),
  setSelectedModules: (paths) => set({ selectedModules: paths }),
  reset: () => {
    const ctrl = get().abortController;
    if (ctrl) {
      ctrl.abort();
    }
    chapterContentBuffer.cancel()
    contentBuffer.cancel()
    set({
      sourceType: null,
      sourceContent: '',
      gitUrl: '',
      scenarioMode: defaultScenarioModeForSource(null),
      modules: null,
      selectedModules: [],
      seriesTitle: '',
      outline: null,
      chapterStatus: {},
      chapterPhases: {},
      chapterErrors: {},
      chapterUsage: {},
      chapterContents: {},
      isScanning: false,
      isAnalyzing: false,
      isGenerating: false,
      analysisStep: -1,
      analysisMessage: '',
      analysisHistory: [],
      resolvedPromptProfile: null,
      classificationStatus: 'idle',
      classificationReason: '',
      progress: '',
      content: '',
      currentChapterTitle: '',
      abortController: null,
      parentBlogId: null,
      currentTaskId: null,
    })
  }
}
})
