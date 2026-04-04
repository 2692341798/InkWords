import { useRef, useCallback } from 'react';
import { fetchEventSource } from '@microsoft/fetch-event-source';
import { useStreamStore } from '../store/streamStore';

export const useBlogStream = () => {
  const { appendContent, setIsStreaming, reset } = useStreamStore();
  const abortControllerRef = useRef<AbortController | null>(null);

  const startStream = useCallback(async (sourceContent: string) => {
    reset();
    setIsStreaming(true);

    abortControllerRef.current = new AbortController();

    try {
      await fetchEventSource('/api/v1/stream/generate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'text/event-stream',
        },
        body: JSON.stringify({ source_content: sourceContent }),
        signal: abortControllerRef.current.signal,
        onmessage(event) {
          if (event.event === 'chunk') {
            try {
              const parsed = JSON.parse(event.data);
              if (typeof parsed === 'string') {
                appendContent(parsed);
              } else if (parsed.content) {
                appendContent(parsed.content);
              } else {
                appendContent(event.data);
              }
            } catch {
              appendContent(event.data);
            }
          } else if (event.event === 'done') {
            setIsStreaming(false);
            if (abortControllerRef.current) {
              abortControllerRef.current.abort();
            }
          }
        },
        onerror(err) {
          console.error('EventSource error:', err);
          setIsStreaming(false);
          throw err; // Stop retrying
        },
        onclose() {
          setIsStreaming(false);
        }
      });
    } catch (err) {
      console.error('Stream failed:', err);
      setIsStreaming(false);
    }
  }, [appendContent, setIsStreaming, reset]);

  const stopStream = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
      setIsStreaming(false);
    }
  }, [setIsStreaming]);

  return { startStream, stopStream };
};
