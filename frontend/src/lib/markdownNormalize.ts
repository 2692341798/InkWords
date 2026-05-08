export function normalizeMarkdown(markdown: string): string {
  const fenceLangs = [
    'cpp',
    'c++',
    'c',
    'go',
    'ts',
    'tsx',
    'js',
    'jsx',
    'bash',
    'sh',
    'python',
    'py',
    'json',
    'yaml',
    'yml',
    'html',
    'css',
    'sql',
    'text',
  ]

  const lines = markdown.replace(/\r\n?/g, '\n').split('\n')
  const out: string[] = []
  let inFence = false

  const normalizeHeadingLine = (line: string) => {
    let next = line
    const spacedHashes = next.match(/^(#(?:\s+#){1,5})\s*(.*)$/)
    if (spacedHashes) {
      const hashCount = (spacedHashes[1].match(/#/g) || []).length
      next = `${'#'.repeat(hashCount)} ${spacedHashes[2]}`
    }
    next = next.replace(/^(#{1,6})(\S)/, '$1 $2')
    next = next.replace(/^(#{1,6})\s*(\d+(?:\.\d+)*)(\.?)(\S)/, (_m, hashes, num, dot, rest) => {
      const maybeDot = dot ? dot : ''
      return `${hashes} ${num}${maybeDot} ${rest}`
    })
    next = next.replace(/^(#{1,6}\s*\d+(?:\.\d+)*\.?)\s+/, '$1 ')
    next = next.replace(/^(#{1,6}\s+\d+(?:\.\d+)*)(?=[^\s.])/g, '$1 ')
    return next
  }

  const normalizeFenceLine = (line: string) => {
    return line.replace(/```([a-zA-Z0-9_+-]+)([^\n])/g, (_m, token: string, next: string) => {
      const lower = token.toLowerCase()
      const prefix = fenceLangs.find((l) => lower === l || lower.startsWith(l))
      if (!prefix) return `\`\`\`${token}\n${next}`
      const rest = token.slice(prefix.length)
      if (!rest) return `\`\`\`${prefix}\n${next}`
      return `\`\`\`${prefix}\n${rest}${next}`
    })
  }

  const splitInlineHeadings = (line: string) => {
    const result: string[] = []
    let cursor = line

    while (true) {
      const match = cursor.match(/(.+?)(#{2,6}\d)/)
      if (!match) break
      const idx = cursor.indexOf(match[2])
      if (idx <= 0) break
      result.push(cursor.slice(0, idx))
      result.push('')
      cursor = cursor.slice(idx)
    }

    result.push(cursor)
    return result
  }

  for (const rawLine of lines) {
    let line = rawLine

    if (line.startsWith('```')) {
      line = normalizeFenceLine(line)
      out.push(line)
      inFence = !inFence
      continue
    }

    if (inFence) {
      out.push(line)
      continue
    }

    const split = splitInlineHeadings(line)
    for (let piece of split) {
      if (!piece) {
        out.push('')
        continue
      }

      piece = piece.replace(/^(\s*[-*+])(\S)/, '$1 $2')

      piece = normalizeHeadingLine(piece)

      const tableGlue = piece.match(/^(#{1,6}.*?)(\|.+)$/)
      if (tableGlue) {
        out.push(normalizeHeadingLine(tableGlue[1].trimEnd()))
        out.push('')
        out.push(tableGlue[2])
        continue
      }

      out.push(piece)
    }
  }

  let joined = out.join('\n')

  joined = joined.replace(/^(#)(?:\s+#){1,5}\s*/gm, (m) => {
    const count = (m.match(/#/g) || []).length
    return `${'#'.repeat(count)} `
  })

  return joined.replace(/([^\n])```/g, '$1\n```')
}
