import { create } from 'zustand';

interface StreamState {
  content: string;
  isStreaming: boolean;
  setContent: (content: string) => void;
  appendContent: (chunk: string) => void;
  setIsStreaming: (isStreaming: boolean) => void;
  reset: () => void;
}

export const useStreamStore = create<StreamState>((set) => ({
  content: '',
  isStreaming: false,
  setContent: (content) => set({ content }),
  appendContent: (chunk) => set((state) => ({ content: state.content + chunk })),
  setIsStreaming: (isStreaming) => set({ isStreaming }),
  reset: () => set({ content: '', isStreaming: false }),
}));
