import type { LearnerCodeFile } from '@/services/mastery'

export function learnerCodeError(files: LearnerCodeFile[]): string | null {
  if (!files.length) return null
  if (files.length > 32) return '最多选择 32 个文件。'
  const paths = new Set<string>()
  let bytes = 0
  for (const file of files) {
    const parts = file.path.split('/')
    if (file.path.length > 200 || !/^[A-Za-z0-9_./-]+$/.test(file.path) || parts.some((part) => !part || part.startsWith('.') || part === 'vendor' || part === 'node_modules') || (file.path !== 'go.mod' && !file.path.endsWith('.go'))) return '文件名需为相对路径的 .go 文件或 go.mod，不接受目录、压缩包或脚本。'
    if (paths.has(file.path)) return '文件路径重复，请调整后再保存。'
    if (files.some((other) => other.path !== file.path && file.path.startsWith(`${other.path}/`))) return '文件路径不能同时用作另一个文件的目录。'
    paths.add(file.path)
    const size = new TextEncoder().encode(file.content).byteLength
    if (!file.content.trim() || file.content.includes('\0') || size > 128 * 1024) return '每个文件需为非空 UTF-8 文本，最多 128 KiB。'
    bytes += size
  }
  if (!files.some((file) => file.path.endsWith('.go'))) return '至少需要一个 .go 源文件。'
  if (bytes > 256 * 1024) return '本次代码总计最多 256 KiB。'
  return null
}

export async function readLearnerCodeFiles(files: File[]): Promise<LearnerCodeFile[]> {
  if (files.length > 32 || files.some((file) => file.size > 128 * 1024) || files.reduce((total, file) => total + file.size, 0) > 256 * 1024) throw new Error('最多 32 个文件，每个 128 KiB，总计 256 KiB。')
  const decoder = new TextDecoder('utf-8', { fatal: true, ignoreBOM: true })
  const loaded = await Promise.all(files.map(async (file) => ({ path: file.name, content: decoder.decode(await file.arrayBuffer()) })))
  const error = learnerCodeError(loaded)
  if (error) throw new Error(error)
  return loaded
}
