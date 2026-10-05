import fs from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'

const frontendRoot = path.resolve(__dirname, '..')
const nginxConfig = fs.readFileSync(path.join(frontendRoot, 'nginx.conf'), 'utf-8')
const viteConfig = fs.readFileSync(path.join(frontendRoot, 'vite.config.ts'), 'utf-8')
const sourceRoot = path.join(frontendRoot, 'src')

const locationBlock = (location: string) => {
  const start = nginxConfig.indexOf(location)
  expect(start).toBeGreaterThanOrEqual(0)
  const end = nginxConfig.indexOf('\n    }', start)
  expect(end).toBeGreaterThan(start)
  return nginxConfig.slice(start, end)
}

describe('frontend microservices gateway contract', () => {
  it('resolves Compose upstreams dynamically so a recreated service cannot leave a stale IP', () => {
    expect(nginxConfig).toContain('resolver 127.0.0.11 ipv6=off valid=10s;')
    expect(nginxConfig).toContain('set $core_api_upstream core-api:8080;')
    expect(nginxConfig).toContain('set $llm_stream_upstream llm-stream:8080;')
    expect(nginxConfig).toContain('set $parser_service_upstream parser-service:8080;')
    expect(nginxConfig).toContain('set $review_service_upstream review-service:8080;')
    expect(nginxConfig).toContain('set $export_service_upstream export-service:8080;')
  })

  it('routes service-owned API paths to the correct upstream before the core fallback', () => {
    const routes = [
      ['location ~ ^/api/v1/tasks/[^/]+/stream$', 'proxy_pass http://$core_api_upstream;'],
      ['location ^~ /api/v1/stream/', 'proxy_pass http://$llm_stream_upstream;'],
      ['location ~ ^/api/v1/blogs/[^/]+/(continue|polish)$', 'proxy_pass http://$llm_stream_upstream;'],
      ['location = /api/v1/project/parse', 'proxy_pass http://$parser_service_upstream;'],
      ['location = /api/v1/project/crawl', 'proxy_pass http://$parser_service_upstream;'],
      ['location ^~ /api/v1/review/', 'proxy_pass http://$review_service_upstream;'],
      ['location ^~ /api/v1/mastery/', 'proxy_pass http://$review_service_upstream;'],
      ['location ~ ^/api/v1/blogs/[^/]+/export', 'proxy_pass http://$export_service_upstream;'],
      ['location ~ ^/api/v1/textbook-projects/chapters/[^/]+/export/(markdown|zip)$', 'proxy_pass http://$export_service_upstream;'],
    ] as const

    const fallbackIndex = nginxConfig.indexOf('location /api/')
    routes.forEach(([location, upstream]) => {
      expect(locationBlock(location)).toContain(upstream)
      expect(nginxConfig.indexOf(location)).toBeLessThan(fallbackIndex)
    })
    expect(locationBlock('location /api/')).toContain('proxy_pass http://$core_api_upstream;')
    expect(locationBlock('location /uploads/')).toContain('proxy_pass http://$core_api_upstream;')
  })

  it('keeps task and legacy streaming routes unbuffered', () => {
    const streamingLocations = [
      'location ~ ^/api/v1/tasks/[^/]+/stream$',
      'location ^~ /api/v1/stream/',
      'location ~ ^/api/v1/blogs/[^/]+/(continue|polish)$',
    ]

    streamingLocations.forEach((location) => {
      const block = locationBlock(location)
      expect(block).toContain('proxy_buffering off;')
      expect(block).toContain('proxy_cache off;')
      expect(block).toContain('add_header X-Accel-Buffering "no";')
      expect(block).toContain('proxy_read_timeout 3600s;')
      expect(block).toContain('proxy_set_header X-Request-ID $http_x_request_id;')
    })
  })

  it('never forwards browser authorization into the fixed local workspace', () => {
    expect(nginxConfig).not.toContain('proxy_set_header Authorization')
    expect(nginxConfig).not.toContain('$http_authorization')
  })

  it('makes Vite proxy only through the configurable gateway origin', () => {
    expect(viteConfig).toContain('INKWORDS_GATEWAY_ORIGIN')
    expect(viteConfig).toContain("'http://localhost:8081'")
    expect(viteConfig).toMatch(/'\/api':\s*\{\s*target: gatewayOrigin/)
    expect(viteConfig).toMatch(/'\/uploads':\s*\{\s*target: gatewayOrigin/)
    expect(viteConfig).not.toContain('core-api:8080')
    expect(viteConfig).not.toContain('llm-stream:8080')
    expect(viteConfig).not.toContain('parser-service:8080')
    expect(viteConfig).not.toContain('export-service:8080')
    expect(viteConfig).not.toContain('review-service:8080')
  })

  it('keeps browser source free from service origins and scattered API literals', () => {
    const sourceFiles = fs.readdirSync(sourceRoot, { recursive: true, withFileTypes: true })
      .filter((entry) => entry.isFile() && /\.(ts|tsx)$/.test(entry.name) && !entry.name.includes('.test.'))
      .map((entry) => path.join(entry.parentPath, entry.name))
    const violations = sourceFiles.flatMap((filePath) => {
      if (filePath.endsWith('services/apiRoutes.ts')) return []
      const content = fs.readFileSync(filePath, 'utf-8')
      const hasServiceOrigin = /https?:\/\/(core-api|llm-stream|parser-service|export-service|review-service)(?::\d+)?/.test(content)
      const hasApiLiteral = /['"`]\/api\//.test(content)
      return hasServiceOrigin || hasApiLiteral ? [path.relative(sourceRoot, filePath)] : []
    })

    expect(violations).toEqual([])
  })
})
