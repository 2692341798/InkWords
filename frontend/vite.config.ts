import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { loadEnv } from 'vite'
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Vite 的 runner config loader 以原生 ESM 执行配置，不能依赖 CommonJS 目录变量。
const frontendRootDir = path.dirname(fileURLToPath(import.meta.url))

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, frontendRootDir, '')
  const gatewayOrigin = process.env.INKWORDS_GATEWAY_ORIGIN
    || env.INKWORDS_GATEWAY_ORIGIN
    || 'http://localhost:8081'

  return {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    plugins: [react(), tailwindcss()] as any,
    resolve: {
      alias: {
        '@': path.resolve(frontendRootDir, './src'),
      },
    },
    server: {
      proxy: {
        '/api': {
          target: gatewayOrigin,
          changeOrigin: true,
          timeout: 3600000,
          proxyTimeout: 3600000,
        },
        '/uploads': {
          target: gatewayOrigin,
          changeOrigin: true,
          timeout: 3600000,
          proxyTimeout: 3600000,
        },
      },
    },
    test: {
      coverage: {
        provider: 'v8',
        reporter: ['text-summary', 'json-summary', 'lcov'],
        include: ['src/**/*.{ts,tsx}'],
        exclude: ['src/**/*.test.{ts,tsx}', 'src/components/ui/**'],
        thresholds: {
          statements: 38,
          branches: 65,
          functions: 54,
          lines: 38,
        },
      },
    },
  }
})
