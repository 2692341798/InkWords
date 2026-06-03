export const STREAM_FLUSH_DELAY_MS = 120

type TimeoutHandle = ReturnType<typeof setTimeout> | null

function createFlushScheduler(flush: () => void, delayMs: number) {
  let timer: TimeoutHandle = null

  const schedule = () => {
    if (timer !== null) {
      return
    }
    timer = setTimeout(() => {
      timer = null
      flush()
    }, delayMs)
  }

  const cancel = () => {
    if (timer !== null) {
      clearTimeout(timer)
      timer = null
    }
  }

  return { schedule, cancel }
}

export function createTextChunkBuffer(
  onFlush: (chunk: string) => void,
  delayMs = STREAM_FLUSH_DELAY_MS,
) {
  let pending = ''
  const scheduler = createFlushScheduler(() => {
    if (!pending) {
      return
    }
    const nextChunk = pending
    pending = ''
    onFlush(nextChunk)
  }, delayMs)

  return {
    push(chunk: string) {
      if (!chunk) {
        return
      }
      pending += chunk
      scheduler.schedule()
    },
    flush() {
      scheduler.cancel()
      if (!pending) {
        return
      }
      const nextChunk = pending
      pending = ''
      onFlush(nextChunk)
    },
    cancel() {
      scheduler.cancel()
      pending = ''
    },
  }
}

export function createChapterChunkBuffer(
  onFlush: (updates: Record<number, string>) => void,
  delayMs = STREAM_FLUSH_DELAY_MS,
) {
  let pending: Record<number, string> = {}
  const scheduler = createFlushScheduler(() => {
    if (Object.keys(pending).length === 0) {
      return
    }
    const nextUpdates = pending
    pending = {}
    onFlush(nextUpdates)
  }, delayMs)

  return {
    push(sort: number, chunk: string) {
      if (!chunk) {
        return
      }
      pending = {
        ...pending,
        [sort]: (pending[sort] ?? '') + chunk,
      }
      scheduler.schedule()
    },
    flush() {
      scheduler.cancel()
      if (Object.keys(pending).length === 0) {
        return
      }
      const nextUpdates = pending
      pending = {}
      onFlush(nextUpdates)
    },
    cancel() {
      scheduler.cancel()
      pending = {}
    },
  }
}
