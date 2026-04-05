import { create } from 'zustand'

export interface Chapter {
  title: string
  summary: string
  sort: number
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
  outline: Chapter[] | null
  chapterStatus: Record<number, 'pending' | 'generating' | 'completed' | 'error'>
  generatedContent: string
  isAnalyzing: boolean
  isGenerating: boolean
  mapReduceProgress: MapReduceProgress | null
  setSource: (type: 'git' | 'file', content: string) => void
  setOutline: (outline: Chapter[]) => void
  updateChapterStatus: (sort: number, status: 'pending' | 'generating' | 'completed' | 'error') => void
  appendGeneratedContent: (chunk: string) => void
  clearGeneratedContent: () => void
  setGenerating: (status: boolean) => void
  setAnalyzing: (status: boolean) => void
  setMapReduceProgress: (progress: MapReduceProgress | null) => void
  reset: () => void
}

export const useStreamStore = create<StreamState>((set) => ({
  sourceType: null,
  sourceContent: '',
  outline: null,
  chapterStatus: {},
  generatedContent: '',
  isAnalyzing: false,
  isGenerating: false,
  mapReduceProgress: null,
  setSource: (type, content) => set({ sourceType: type, sourceContent: content }),
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
  reset: () => set({
    sourceType: null,
    sourceContent: '',
    outline: null,
    chapterStatus: {},
    generatedContent: '',
    isAnalyzing: false,
    isGenerating: false,
    mapReduceProgress: null
  })
}))
