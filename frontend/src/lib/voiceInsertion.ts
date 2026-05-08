export function replaceVoiceSegment(params: {
  base: string
  anchorStart: number
  anchorEnd: number
  lastInsertedLength: number
  nextText: string
}) {
  const { base, anchorStart, anchorEnd, lastInsertedLength, nextText } = params
  const start = anchorStart
  const effectiveAnchorEnd = lastInsertedLength > 0 ? start : anchorEnd
  const previousEnd = anchorStart + lastInsertedLength
  const afterStart = Math.max(effectiveAnchorEnd, previousEnd)

  const before = base.slice(0, start)
  const after = base.slice(afterStart)
  const merged = `${before}${nextText}${after}`

  return { merged, newInsertedLength: nextText.length }
}
