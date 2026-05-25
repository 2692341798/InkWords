export function extractPolishedBody(draft: string): string {
  const separator = '\n---\n'
  const idx = draft.indexOf(separator)
  const body = idx === -1 ? draft : draft.slice(idx + separator.length)
  return sanitizeLeadingGeneratedMarkdown(body)
}

function sanitizeLeadingGeneratedMarkdown(content: string): string {
  const cleaned = content.replace(/<think>[\s\S]*?<\/think>/g, '').trim()
  if (!cleaned) return ''

  const paragraphs = cleaned.split(/\n\s*\n/)
  let startIndex = 0
  while (startIndex < paragraphs.length && isLeadingMetaParagraph(paragraphs[startIndex])) {
    startIndex += 1
  }

  return paragraphs.slice(startIndex).join('\n\n').trim()
}

function isLeadingMetaParagraph(paragraph: string): boolean {
  const trimmed = paragraph.trim()
  if (!trimmed) return false
  if (/^(#{1,4}\s|>\s|- |\* |```|\||\d+\.\s)/.test(trimmed)) {
    return false
  }

  const hasRoleIntro =
    trimmed.includes('作为高级') ||
    trimmed.includes('作为一名') ||
    (trimmed.includes('作为') &&
      ['架构师', '博主', '助手', 'AI'].some(keyword => trimmed.includes(keyword)))

  const hasTaskTalk =
    trimmed.includes('收到你的需求') ||
    trimmed.includes('根据你提供') ||
    trimmed.includes('我将根据') ||
    trimmed.includes('我会根据') ||
    trimmed.includes('以下是根据') ||
    trimmed.includes('我将为你') ||
    trimmed.includes('接下来我将') ||
    trimmed.includes('下面我将')

  const hasGenerationIntent =
    trimmed.includes('撰写') ||
    trimmed.includes('生成') ||
    trimmed.includes('整理') ||
    trimmed.includes('输出') ||
    trimmed.includes('博客') ||
    trimmed.includes('文章')

  return hasRoleIntro || (hasTaskTalk && hasGenerationIntent)
}
