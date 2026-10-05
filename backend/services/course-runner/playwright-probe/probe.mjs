import { chromium } from 'playwright'
import { spawn } from 'node:child_process'
import { existsSync } from 'node:fs'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)

const MAX_CONSOLE_ENTRIES = 64
const MAX_NETWORK_ENTRIES = 128
const MAX_SCREENSHOT_BYTES = 512 * 1024

function argument(name) {
  const index = process.argv.indexOf(name)
  if (index < 0 || index + 1 >= process.argv.length) throw new Error(`missing ${name}`)
  return process.argv[index + 1]
}

function localBaseURL(value) {
  const parsed = new URL(value)
  if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1' || !parsed.port || parsed.pathname !== '/' || parsed.search || parsed.hash) {
    throw new Error('base URL must be an explicit 127.0.0.1 HTTP origin')
  }
  return parsed
}

function localPath(value) {
  if (!/^\/[A-Za-z0-9._/-]{1,255}$/.test(value) || value.includes('//') || value.includes('..')) {
    throw new Error('page path is not a bounded local path')
  }
  return value
}

function expectedText(value) {
  if (!value.trim() || value.length > 256 || /[\u0000-\u001f\u007f]/.test(value)) throw new Error('expected text is invalid')
  return value
}

function redact(value) {
  return String(value)
    .replace(/(authorization\s*:\s*bearer\s+)[^\s,;]+/ig, '$1[REDACTED]')
    .replace(/\b((?:api[_-]?key|access[_-]?token|refresh[_-]?token|password|secret)\s*[:=]\s*)[^\s,;]+/ig, '$1[REDACTED]')
    .slice(0, 2048)
}

function compactFailure(error) {
  const value = String(error?.message || 'unknown error')
  if (/No usable sandbox|Chromium sandboxing failed/i.test(value)) return 'chromium_sandbox_unavailable'
  if (/teaching page did not become ready/i.test(value)) return 'teaching_page_start_timeout'
  if (/expected text was not found/i.test(value)) return 'expected_text_missing'
  return redact(value)
}

function requireChromiumSandboxKernelInterface() {
  // The container forbids setuid escalation, so Chromium must create its own
  // user namespace. Without the proc uid-map interface that sandbox cannot be
  // established; fail before starting the generated page or browser process.
  if (!existsSync('/proc/self/uid_map')) throw new Error('chromium_sandbox_unavailable: proc_uid_map_unavailable')
}

async function main() {
  requireChromiumSandboxKernelInterface()
  const baseURL = localBaseURL(argument('--base-url'))
  const path = localPath(argument('--path'))
  const text = expectedText(argument('--expected-text'))
  const pageURL = new URL(path, baseURL).toString()
  const consoleEntries = []
  const network = []

  const server = spawn('go', ['run', '.'], {
    cwd: '/workspace',
    env: { ...process.env, GOPROXY: 'off', GOSUMDB: 'off' },
    stdio: ['ignore', 'ignore', 'ignore'],
  })
  try {
    await waitForTeachingPage(pageURL)
    await capturePage(baseURL, path, text, pageURL, consoleEntries, network)
  } finally {
    server.kill('SIGTERM')
  }
}

async function capturePage(baseURL, path, text, pageURL, consoleEntries, network) {
  const browser = await chromium.launch({
    headless: true,
    // Keep Chromium's own process sandbox mandatory even inside Bubblewrap;
    // the runtime must fail closed instead of adding a disabling launch flag.
    chromiumSandbox: true,
    // DNS and browser background services are disabled in addition to the
    // outer Bubblewrap network namespace. The page router below remains the
    // final application-level boundary for every request.
    args: [
      '--disable-background-networking',
      '--disable-component-update',
      '--disable-default-apps',
      '--disable-sync',
      '--host-resolver-rules=MAP * ~NOTFOUND, EXCLUDE 127.0.0.1',
      '--no-first-run',
    ],
  })
  try {
    const browserVersion = browser.version()
    const context = await browser.newContext({ serviceWorkers: 'block' })
    const page = await context.newPage()
    await page.route('**/*', async (route) => {
      const requested = new URL(route.request().url())
      if (requested.protocol !== 'http:' || requested.hostname !== baseURL.hostname || requested.port !== baseURL.port) {
        await route.abort('blockedbyclient')
        return
      }
      await route.continue()
    })
    page.on('console', (message) => {
      if (consoleEntries.length < MAX_CONSOLE_ENTRIES) consoleEntries.push({ type: message.type(), text: redact(message.text()) })
    })
    page.on('response', (response) => {
      const responseURL = new URL(response.url())
      if (responseURL.protocol === 'http:' && responseURL.hostname === baseURL.hostname && responseURL.port === baseURL.port && network.length < MAX_NETWORK_ENTRIES) {
        network.push({ url: response.url(), resource_type: response.request().resourceType(), status: response.status() })
      }
    })

    const response = await page.goto(pageURL, { waitUntil: 'networkidle', timeout: 15_000 })
    if (!response || response.status() < 200 || response.status() > 299) throw new Error('teaching page did not return a successful HTTP response')
    const target = page.getByText(text, { exact: false })
    if (await target.count() === 0) throw new Error('expected text was not found on the teaching page')
    const screenshot = await page.screenshot({ type: 'png', fullPage: true })
    if (screenshot.length === 0 || screenshot.length > MAX_SCREENSHOT_BYTES) throw new Error('teaching-page screenshot is unavailable or exceeds the evidence budget')
    process.stdout.write(`${JSON.stringify({
      url: pageURL,
      final_url: page.url(),
      browser_name: 'Chromium',
      browser_version: browserVersion,
      screenshot_base64: screenshot.toString('base64'),
      dom_assertions: [{ locator: `text=${text}`, assertion: 'has_text', expected: text }],
      console: consoleEntries,
      network,
      playwright_version: requirePlaywrightVersion(),
    })}\n`)
  } finally {
    await browser.close()
  }
}

async function waitForTeachingPage(pageURL) {
  // A cold, single-worker Go cache can take slightly over ten seconds to
  // compile net/http under the fixed memory limit.
  const deadline = Date.now() + 20_000
  while (Date.now() < deadline) {
    try {
      const response = await fetch(pageURL)
      if (response.ok) return
    } catch {
      // The generated Go binary is still compiling or binding loopback.
    }
    await new Promise((resolve) => setTimeout(resolve, 100))
  }
  throw new Error('loopback teaching page did not become ready')
}

function requirePlaywrightVersion() {
  // Package metadata is operator-owned image content, not task input. Reading
  // it at runtime prevents the evidence record from drifting when the pinned
  // Playwright image and lockfile are deliberately upgraded together.
  const version = require('playwright/package.json')?.version
  if (typeof version !== 'string' || !version.trim()) throw new Error('Playwright version is unavailable')
  return version
}

main().catch((error) => {
  process.stderr.write(`browser probe failed: ${compactFailure(error)}\n`)
  process.exitCode = 1
})
