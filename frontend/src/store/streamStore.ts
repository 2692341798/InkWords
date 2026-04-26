import { create } from 'zustand'

export interface Chapter {
  id?: string
  title: string
  summary: string
  sort: number
  files?: string[]
  action?: 'new' | 'regenerate' | 'skip' | string
}

export interface MapReduceProgress {
  status: 'chunk_analyzing' | 'chunk_done' | 'chunk_failed' | 'chunk_failed_final' | ''
  dir: string
  index: number
  total: number
  attempt?: number
  worker_id?: number
}

export interface ModuleCard {
  path: string
  name: string
  description: string
}

interface StreamState {
  sourceType: 'git' | 'file' | null
  sourceContent: string
  gitUrl: string
  modules: ModuleCard[] | null
  selectedModules: string[]
  seriesTitle: string
  outline: Chapter[] | null
  chapterStatus: Record<number, 'pending' | 'generating' | 'completed' | 'error'>
  generatedContent: string
  chapterContents: Record<number, string>
  isScanning: boolean
  isAnalyzing: boolean
  isGenerating: boolean
  mapReduceProgress: MapReduceProgress | null
  workers: Record<number, MapReduceProgress>
  analysisStep: number
  analysisMessage: string
  progress: string
  content: string
  currentChapterTitle: string
  abortController: AbortController | null
  parentBlogId: string | null
  setSource: (type: 'git' | 'file', content: string, gitUrl?: string) => void
  setSourceContent: (content: string) => void
  setSeriesTitle: (title: string) => void
  setOutline: (outline: Chapter[]) => void
  updateChapter: (sort: number, field: 'title' | 'summary', value: string) => void
  addChapter: () => void
  removeChapter: (sort: number) => void
  moveChapter: (sort: number, direction: 'up' | 'down') => void
  updateChapterStatus: (sort: number, status: 'pending' | 'generating' | 'completed' | 'error') => void
  appendGeneratedContent: (chunk: string) => void
  appendChapterContent: (sort: number, content: string) => void
  appendChapterContents: (updates: Record<number, string>) => void
  clearGeneratedContent: () => void
  clearChapterContent: (sort: number) => void
  setScanning: (status: boolean) => void
  setGenerating: (status: boolean) => void
  setAnalyzing: (status: boolean) => void
  setMapReduceProgress: (progress: MapReduceProgress | null) => void
  setAnalysisStep: (step: number) => void
  setAnalysisMessage: (msg: string) => void
  setProgress: (msg: string) => void
  setContent: (content: string) => void
  appendContent: (chunk: string) => void
  setCurrentChapterTitle: (title: string) => void
  setAbortController: (ctrl: AbortController | null) => void
  setParentBlogId: (id: string | null) => void
  setModules: (modules: ModuleCard[] | null) => void
  setSelectedModules: (paths: string[]) => void
  stopAllStreams: () => void
  reset: () => void
}

export const useStreamStore = create<StreamState>((set, get) => ({
  sourceType: null,
  sourceContent: '',
  gitUrl: '',
  modules: null,
  selectedModules: [],
  seriesTitle: '',
  outline: null,
  chapterStatus: {},
  generatedContent: '',
  chapterContents: {},
  isScanning: false,
  isAnalyzing: false,
  isGenerating: false,
  mapReduceProgress: null,
  workers: {},
  analysisStep: -1,
  analysisMessage: '',
  progress: '',
  content: '',
  currentChapterTitle: '',
  abortController: null,
  parentBlogId: null,
  setSource: (type, content, gitUrl) => set({ sourceType: type, sourceContent: content, gitUrl: gitUrl || '' }),
  setSourceContent: (content) => set({ sourceContent: content }),
  setSeriesTitle: (title) => set({ seriesTitle: title }),
  setOutline: (outline) => set({ 
    outline,
    chapterStatus: outline.reduce((acc, ch) => ({ ...acc, [ch.sort]: 'pending' }), {}),
    chapterContents: outline.reduce((acc, ch) => ({ ...acc, [ch.sort]: '' }), {})
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
    const newContents = { ...state.chapterContents };
    delete newStatus[sort];
    delete newContents[sort];
    return { outline: newOutline, chapterStatus: newStatus, chapterContents: newContents };
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
  appendGeneratedContent: (chunk) => set((state) => ({ generatedContent: state.generatedContent + chunk })),
  appendChapterContent: (sort, chunk) =>
    set((state) => ({
      chapterContents: {
        ...state.chapterContents,
        [sort]: (state.chapterContents[sort] || '') + chunk
      }
    })),
  appendChapterContents: (updates) =>
    set((state) => {
      const newContents = { ...state.chapterContents };
      for (const [sort, chunk] of Object.entries(updates)) {
        const key = Number(sort);
        newContents[key] = (newContents[key] || '') + chunk;
      }
      return { chapterContents: newContents };
    }),
  clearGeneratedContent: () => set({ generatedContent: '' }),
  clearChapterContent: (sort) =>
    set((state) => ({
      chapterContents: {
        ...state.chapterContents,
        [sort]: ''
      }
    })),
  setScanning: (status) => set({ isScanning: status }),
  setGenerating: (status) => set({ isGenerating: status }),
  setAnalyzing: (status) => set({ isAnalyzing: status }),
  setMapReduceProgress: (progress) => set((state) => {
    if (!progress) return { mapReduceProgress: null, workers: {} };
    if (progress.worker_id !== undefined) {
      return {
        mapReduceProgress: progress,
        workers: {
          ...state.workers,
          [progress.worker_id]: progress
        }
      }
    }
    return { mapReduceProgress: progress }
  }),
  setAnalysisStep: (step) => set({ analysisStep: step }),
  setAnalysisMessage: (msg) => set({ analysisMessage: msg }),
  setProgress: (msg) => set({ progress: msg }),
  setContent: (content) => set({ content }),
  appendContent: (chunk) => set((state) => ({ content: state.content + chunk })),
  setCurrentChapterTitle: (title) => set({ currentChapterTitle: title }),
  setAbortController: (ctrl) => set({ abortController: ctrl }),
  setParentBlogId: (id) => set({ parentBlogId: id }),
  setModules: (modules) => set({ modules }),
  setSelectedModules: (paths) => set({ selectedModules: paths }),
  stopAllStreams: () => {
    const ctrl = get().abortController;
    if (ctrl) {
      ctrl.abort();
    }
    set((state) => {
      const newStatus = { ...state.chapterStatus };
      Object.keys(newStatus).forEach((key) => {
        if (newStatus[Number(key)] === 'generating') {
          newStatus[Number(key)] = 'pending';
        }
      });
      return { 
        isScanning: false,
        isAnalyzing: false, 
        isGenerating: false, 
        analysisStep: -1, 
        abortController: null,
        chapterStatus: newStatus
      };
    });
  },
  reset: () => {
    const ctrl = get().abortController;
    if (ctrl) {
      ctrl.abort();
    }
    set({
      sourceType: null,
      sourceContent: '',
      gitUrl: '',
      modules: null,
      selectedModules: [],
      seriesTitle: '',
      outline: null,
      chapterStatus: {},
      generatedContent: '',
      chapterContents: {},
      isScanning: false,
      isAnalyzing: false,
      isGenerating: false,
      mapReduceProgress: null,
      workers: {},
      analysisStep: -1,
      analysisMessage: '',
      progress: '',
      content: '',
      currentChapterTitle: '',
      abortController: null,
      parentBlogId: null
    })
  }
}))
