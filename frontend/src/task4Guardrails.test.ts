import fs from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'

type ScanResult = {
  filePath: string
  message: string
}

const srcRoot = path.resolve(__dirname)

const collectSourceFiles = (rootDir: string) => {
  const results: string[] = []
  const entries = fs.readdirSync(rootDir, { withFileTypes: true })
  entries.forEach((entry) => {
    const fullPath = path.join(rootDir, entry.name)
    if (entry.isDirectory()) {
      results.push(...collectSourceFiles(fullPath))
      return
    }

    if (!fullPath.endsWith('.ts') && !fullPath.endsWith('.tsx')) return
    if (fullPath.endsWith('.test.ts') || fullPath.endsWith('.test.tsx')) return
    results.push(fullPath)
  })
  return results
}

const scanForbiddenInteractions = (filePath: string, fileContent: string): ScanResult[] => {
  const violations: ScanResult[] = []

  const patterns: Array<{ name: string; regex: RegExp }> = [
    { name: 'alert', regex: /\balert\s*\(/ },
    { name: 'confirm', regex: /\b(?:window\.)?confirm\s*\(/ },
    { name: 'location.reload', regex: /\b(?:window\.)?location\.reload\s*\(/ },
  ]

  patterns.forEach((pattern) => {
    if (pattern.regex.test(fileContent)) {
      violations.push({ filePath, message: `包含禁止交互：${pattern.name}` })
    }
  })

  const hrefRegex = /\b(?:window\.)?location\.href\s*=/
  if (hrefRegex.test(fileContent)) {
    const lines = fileContent.split(/\r?\n/)

    lines.forEach((line, index) => {
      if (!hrefRegex.test(line)) return
      violations.push({
        filePath,
        message: `包含禁止交互：location.href (第 ${index + 1} 行)`,
      })
    })
  }

  return violations
}

describe('Task 4 guardrails', () => {
  it('removes alert/confirm/location.reload/location.href from frontend/src', () => {
    const files = collectSourceFiles(srcRoot)
    const violations = files.flatMap((filePath) =>
      scanForbiddenInteractions(filePath, fs.readFileSync(filePath, 'utf-8')),
    )

    expect(violations).toEqual([])
  })

  it('keeps the local App and Sidebar free of login gates', () => {
    const appPath = path.resolve(srcRoot, './App.tsx')
    const appContent = fs.readFileSync(appPath, 'utf-8')
    const sidebarPath = path.resolve(srcRoot, './components/Sidebar.tsx')
    const sidebarContent = fs.readFileSync(sidebarPath, 'utf-8')

    expect(appContent).not.toContain('authTokenStore')
    expect(appContent).not.toContain('isAuthBypassEnabled')
    expect(appContent).not.toContain("@/pages/Login")
    expect(sidebarContent).not.toContain('authTokenStore')
    expect(sidebarContent).not.toContain('退出登录')
  })

  it('keeps the local single-user UI free of account profile surfaces', () => {
    expect(fs.existsSync(path.resolve(srcRoot, './pages/Dashboard.tsx'))).toBe(false)
    expect(fs.existsSync(path.resolve(srcRoot, './services/user.ts'))).toBe(false)

    const appContent = fs.readFileSync(path.resolve(srcRoot, './App.tsx'), 'utf-8')
    const sidebarContent = fs.readFileSync(path.resolve(srcRoot, './components/Sidebar.tsx'), 'utf-8')
    expect(appContent).not.toContain("currentView === 'dashboard'")
    expect(sidebarContent).not.toContain('个人中心')
  })

  it('uses deferred preview content to reduce Markdown preview re-render cost', () => {
    const editorBodyPath = path.resolve(srcRoot, './components/editor/EditorBody.tsx')
    const content = fs.readFileSync(editorBodyPath, 'utf-8')

    expect(content).toContain('useDeferredValue')
    expect(content).toMatch(/MarkdownEngine\s+content=\{\s*activePreviewTab\s*===\s*'polish'\s*\?\s*deferred/i)
    expect(content).not.toMatch(/MarkdownEngine\s+content=\{[^}]*:\s*content\s*\}/)
  })
})
