const editorOwnerStorageKey = 'inkwords:textbook-editor-owner:v1'
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

let memoryOwnerID: string | null = null

const newOwnerID = () => {
  if (!globalThis.crypto?.randomUUID) {
    throw new Error('当前浏览器无法生成本地编辑会话 ID，请升级浏览器后重试。')
  }
  return globalThis.crypto.randomUUID()
}

/** Returns a stable, browser-local UUID used only to own a short chapter edit lease. */
export function getChapterEditorOwnerID() {
  if (memoryOwnerID) return memoryOwnerID
  try {
    const stored = localStorage.getItem(editorOwnerStorageKey)
    if (stored && uuidPattern.test(stored)) {
      memoryOwnerID = stored
      return stored
    }
    const ownerID = newOwnerID()
    localStorage.setItem(editorOwnerStorageKey, ownerID)
    memoryOwnerID = ownerID
    return ownerID
  } catch {
    memoryOwnerID = newOwnerID()
    return memoryOwnerID
  }
}

/** Hashes the exact Markdown that is persisted in an immutable chapter revision. */
export async function hashChapterMarkdown(markdown: string) {
  if (!globalThis.crypto?.subtle) {
    throw new Error('当前浏览器无法计算内容校验值，无法安全保存章节。')
  }
  const digest = await globalThis.crypto.subtle.digest('SHA-256', new TextEncoder().encode(markdown))
  return [...new Uint8Array(digest)].map((byte) => byte.toString(16).padStart(2, '0')).join('')
}
