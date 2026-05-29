import fs from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'

type FrontendPackageJson = {
  scripts?: Record<string, string>
}

const readPackageJson = (): FrontendPackageJson => {
  const packageJsonPath = path.resolve(__dirname, '../package.json')
  return JSON.parse(fs.readFileSync(packageJsonPath, 'utf-8')) as FrontendPackageJson
}

const readViteConfig = () => {
  const viteConfigPath = path.resolve(__dirname, '../vite.config.ts')
  return fs.readFileSync(viteConfigPath, 'utf-8')
}

describe('frontend toolchain permission compatibility', () => {
  it('uses the runner config loader for vite and vitest commands', () => {
    const scripts = readPackageJson().scripts ?? {}

    expect(scripts.dev).toContain('--configLoader runner')
    expect(scripts.build).toContain('--configLoader runner')
    expect(scripts.preview).toContain('--configLoader runner')
    expect(scripts.test).toContain('--configLoader runner')
  })

  it('keeps vite config path resolution compatible with native esm execution', () => {
    const viteConfig = readViteConfig()

    expect(viteConfig).toContain('fileURLToPath(import.meta.url)')
    expect(viteConfig).not.toContain('__dirname')
  })
})
