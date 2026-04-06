import { create } from 'zustand'

export interface Chapter {
  title: string
  summary: string
  sort: number
  files?: string[]
}

export interface MapReduceProgress {
  status: 'chunk_analyzing' | 'chunk_done' | 'chunk_failed' | 'chunk_failed_final' | ''
  dir: string
  index: number
  total: number
  attempt?: number
  worker_id?: number
}

interface StreamState {
  sourceType: 'git' | 'file' | null
  sourceContent: string
  gitUrl: string
  seriesTitle: string
  outline: Chapter[] | null
  chapterStatus: Record<number, 'pending' | 'generating' | 'completed' | 'error'>
  generatedContent: string
  isAnalyzing: boolean
  isGenerating: boolean
  mapReduceProgress: MapReduceProgress | null
  workers: Record<number, MapReduceProgress>
  analysisStep: number
  analysisMessage: string
  abortController: AbortController | null
  setSource: (type: 'git' | 'file', content: string, gitUrl?: string) => void
  setSeriesTitle: (title: string) => void
  setOutline: (outline: Chapter[]) => void
  updateChapter: (sort: number, field: 'title' | 'summary', value: string) => void
  addChapter: () => void
  removeChapter: (sort: number) => void
  moveChapter: (sort: number, direction: 'up' | 'down') => void
  updateChapterStatus: (sort: number, status: 'pending' | 'generating' | 'completed' | 'error') => void
  appendGeneratedContent: (chunk: string) => void
  clearGeneratedContent: () => void
  setGenerating: (status: boolean) => void
  setAnalyzing: (status: boolean) => void
  setMapReduceProgress: (progress: MapReduceProgress | null) => void
  setAnalysisStep: (step: number) => void
  setAnalysisMessage: (msg: string) => void
  setAbortController: (ctrl: AbortController | null) => void
  stopAllStreams: () => void
  reset: () => void
}

export const useStreamStore = create<StreamState>((set, get) => ({
  sourceType: null,
  sourceContent: '',
  gitUrl: '',
  seriesTitle: '',
  outline: null,
  chapterStatus: {},
  generatedContent: '',
  isAnalyzing: false,
  isGenerating: false,
  mapReduceProgress: null,
  workers: {},
  analysisStep: -1,
  analysisMessage: '',
  abortController: null,
  setSource: (type, content, gitUrl) => set({ sourceType: type, sourceContent: content, gitUrl: gitUrl || '' }),
  setSeriesTitle: (title) => set({ seriesTitle: title }),
  setOutline: (outline) => set({ 
    outline,
    chapterStatus: outline.reduce((acc, ch) => ({ ...acc, [ch.sort]: 'pending' }), {})
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
      files: []
    };
    return { outline: [...state.outline, newChapter] };
  }),
  removeChapter: (sort) => set((state) => {
    if (!state.outline) return state;
    const newOutline = state.outline
      .filter(ch => ch.sort !== sort)
      .map((ch, index) => ({ ...ch, sort: index + 1 }));
    return { outline: newOutline };
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
  clearGeneratedContent: () => set({ generatedContent: '' }),
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
  setAbortController: (ctrl) => set({ abortController: ctrl }),
  stopAllStreams: () => {
    const ctrl = get().abortController;
    if (ctrl) {
      ctrl.abort();
    }
    set({ 
      isAnalyzing: false, 
      isGenerating: false, 
      analysisStep: -1, 
      abortController: null 
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
      seriesTitle: '',
      outline: null,
      chapterStatus: {},
      generatedContent: '',
      isAnalyzing: false,
      isGenerating: false,
      mapReduceProgress: null,
      workers: {},
      analysisStep: -1,
      analysisMessage: '',
      abortController: null
    })
  }
}))
