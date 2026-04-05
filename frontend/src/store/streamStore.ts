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
}

interface StreamState {
  sourceType: 'git' | 'file' | null
  sourceContent: string
  gitUrl: string
  outline: Chapter[] | null
  chapterStatus: Record<number, 'pending' | 'generating' | 'completed' | 'error'>
  generatedContent: string
  isAnalyzing: boolean
  isGenerating: boolean
  mapReduceProgress: MapReduceProgress | null
  analysisStep: number
  analysisMessage: string
  abortController: AbortController | null
  setSource: (type: 'git' | 'file', content: string, gitUrl?: string) => void
  setOutline: (outline: Chapter[]) => void
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
  outline: null,
  chapterStatus: {},
  generatedContent: '',
  isAnalyzing: false,
  isGenerating: false,
  mapReduceProgress: null,
  analysisStep: -1,
  analysisMessage: '',
  abortController: null,
  setSource: (type, content, gitUrl) => set({ sourceType: type, sourceContent: content, gitUrl: gitUrl || '' }),
  setOutline: (outline) => set({ 
    outline,
    chapterStatus: outline.reduce((acc, ch) => ({ ...acc, [ch.sort]: 'pending' }), {})
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
  setMapReduceProgress: (progress) => set({ mapReduceProgress: progress }),
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
      outline: null,
      chapterStatus: {},
      generatedContent: '',
      isAnalyzing: false,
      isGenerating: false,
      mapReduceProgress: null,
      analysisStep: -1,
      analysisMessage: '',
      abortController: null
    })
  }
}))
