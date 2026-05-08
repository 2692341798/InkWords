export function extractPolishedBody(draft: string): string {
  const separator = '\n---\n'
  const idx = draft.indexOf(separator)
  if (idx === -1) return draft.trim()
  return draft.slice(idx + separator.length).trim()
}

