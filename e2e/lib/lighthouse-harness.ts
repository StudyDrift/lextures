/**
 * Shared helpers for the authenticated Lighthouse harness (LH.1).
 * Used by e2e/scripts/run-lighthouse-dashboard.ts and lighthouse-harness.spec.ts.
 */
import { createServer } from 'node:net'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { tmpdir } from 'node:os'
import { fileURLToPath } from 'node:url'

import { chromium, type BrowserContext, type Page } from '@playwright/test'
import lighthouse, { type Result } from 'lighthouse'

import {
  apiCreateCourse,
  apiCreateModule,
  apiEnroll,
  apiSignup,
} from '../fixtures/api.js'
import { uniqueEmail } from '../fixtures/test.js'

export const ACCESS_TOKEN_STORAGE_KEY = 'studydrift_access_token'
export const UI_THEME_STORAGE_KEY = 'lextures.uiTheme'

export type UiTheme = 'light' | 'dark'

export interface LighthouseHarnessOptions {
  pageUrl: string
  theme: UiTheme
  outputPath: string
  token?: string
  /** When true, fail immediately if no token is supplied (AC-3). */
  requireAuth?: boolean
  allowNonLocalhost?: boolean
}

export interface LighthouseHarnessResult {
  outputPath: string
  performanceScore: number
  accessibilityScore: number
  requestedUrl: string
}

const __dirname = dirname(fileURLToPath(import.meta.url))
export const REPO_ROOT = resolve(__dirname, '../..')
export const DEFAULT_OUTPUT_PATH = join(REPO_ROOT, 'docs/lighthouse/global-dashboard-darkmode.json')

/** Refuses non-localhost targets unless explicitly overridden (security). */
export function assertLocalhostOrigin(pageUrl: string, allowNonLocalhost = false): void {
  if (allowNonLocalhost) return
  let parsed: URL
  try {
    parsed = new URL(pageUrl)
  } catch {
    throw new Error(`Invalid PAGE_URL: ${pageUrl}`)
  }
  const host = parsed.hostname.toLowerCase()
  if (host !== 'localhost' && host !== '127.0.0.1' && host !== '[::1]') {
    throw new Error(
      `Lighthouse harness only runs against localhost by default (got ${parsed.origin}). ` +
        'Set LH_ALLOW_NON_LOCALHOST=1 to override.',
    )
  }
}

/** Browser init script payload — mirrors injectToken + applyUiTheme('dark'). */
export function buildAuthThemeInitPayload(token: string, theme: UiTheme) {
  return {
    token,
    theme,
    accessKey: ACCESS_TOKEN_STORAGE_KEY,
    themeKey: UI_THEME_STORAGE_KEY,
  }
}

/** Init script registered on the persistent context before any navigation. */
export function registerAuthThemeInitScript(
  context: BrowserContext,
  token: string,
  theme: UiTheme,
): Promise<void> {
  const payload = buildAuthThemeInitPayload(token, theme)
  return context.addInitScript((data) => {
    localStorage.setItem(data.accessKey, data.token)
    localStorage.setItem(data.themeKey, data.theme)
    localStorage.setItem('lextures-search-shortcut-tip-dismissed', '1')
    localStorage.setItem(
      'lextures.onboarding.v1',
      JSON.stringify({ student: true, teacher: true, admin: true }),
    )
    const root = document.documentElement
    root.classList.toggle('dark', data.theme === 'dark')
    root.style.colorScheme = data.theme === 'dark' ? 'dark' : 'light'
  }, payload)
}

export async function clearIndexedDb(page: Page): Promise<void> {
  await page.evaluate(async () => {
    if (typeof indexedDB.databases !== 'function') return
    try {
      const databases = await indexedDB.databases()
      await Promise.all(
        databases.map(
          (db) =>
            new Promise<void>((resolvePromise, reject) => {
              const request = indexedDB.deleteDatabase(db.name ?? '')
              request.onsuccess = () => resolvePromise()
              request.onerror = () => reject(request.error)
              request.onblocked = () => resolvePromise()
            }),
        ),
      )
    } catch {
      /* ignore — some origins or embedded contexts block IndexedDB */
    }
  })
}

/** Wait until the authenticated dashboard shell is ready (FR-3). */
export async function waitForDashboardReady(page: Page): Promise<void> {
  const mainNav = page.getByRole('navigation', { name: 'Main' })
  await mainNav.waitFor({ state: 'visible', timeout: 30_000 })
  await page.getByText('Loading your dashboard.').waitFor({ state: 'hidden', timeout: 30_000 })
}

export async function assertDocumentTheme(page: Page, theme: UiTheme): Promise<void> {
  const hasDark = await page.evaluate(() => document.documentElement.classList.contains('dark'))
  if (theme === 'dark' && !hasDark) {
    throw new Error('Expected document.documentElement to have class "dark" at audit time')
  }
  if (theme === 'light' && hasDark) {
    throw new Error('Expected document.documentElement to not have class "dark" at audit time')
  }
}

export async function seedLighthouseDashboardUser(theme: UiTheme = 'dark'): Promise<{ token: string; email: string }> {
  const email = uniqueEmail('lh-dash')
  const password = 'E2eTestPass1!'
  const { access_token: token } = await apiSignup({ email, password })

  const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
  const themeRes = await fetch(`${apiBase}/api/v1/settings/account`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ uiTheme: theme }),
  })
  if (!themeRes.ok) {
    const body = await themeRes.text()
    throw new Error(`Patch uiTheme failed (${themeRes.status}): ${body}`)
  }

  for (const title of ['LH Dashboard Course A', 'LH Dashboard Course B']) {
    const course = await apiCreateCourse(token, { title })
    await apiEnroll(token, course.courseCode, email, 'teacher')
    await apiCreateModule(token, course.courseCode, 'Unit 1')
  }

  return { token, email }
}

export function resolveAuthToken(options: LighthouseHarnessOptions): string {
  if (options.requireAuth && !options.token) {
    throw new Error('auth required: supply LH_TOKEN or run without LH_REQUIRE_AUTH=1')
  }
  if (!options.token) {
    throw new Error('auth required: no token available after seeding')
  }
  return options.token
}

async function getFreePort(): Promise<number> {
  return new Promise((resolvePort, reject) => {
    const server = createServer()
    server.listen(0, () => {
      const address = server.address()
      if (!address || typeof address === 'string') {
        reject(new Error('Could not allocate a free port'))
        return
      }
      const { port } = address
      server.close((err) => (err ? reject(err) : resolvePort(port)))
    })
    server.on('error', reject)
  })
}

function lighthouseConfig(theme: UiTheme) {
  return {
    extends: 'lighthouse:default',
    settings: {
      onlyCategories: ['performance', 'accessibility'],
      formFactor: 'mobile' as const,
      locale: 'en-US',
      throttlingMethod: 'simulate' as const,
      screenEmulation: {
        mobile: true,
        width: 412,
        height: 823,
        deviceScaleFactor: 1.75,
        disabled: false,
      },
      emulatedUserAgent:
        'Mozilla/5.0 (Linux; Android 11; moto g power (2022)) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Mobile Safari/537.36',
      disableStorageReset: true,
      // Clear IndexedDB ourselves before navigation; keep localStorage for auth/theme.
      clearStorageTypes: ['cache_storage', 'service_workers'],
      skipAboutBlank: true,
      // Extra guard so UiThemeSync does not flip theme mid-audit.
      ...(theme === 'dark' ? { extraHeaders: {} } : {}),
    },
  }
}

function validateReport(report: Result, outputPath: string): LighthouseHarnessResult {
  if (report.runtimeError) {
    throw new Error(
      `Lighthouse runtime error (${report.runtimeError.code}): ${report.runtimeError.message}`,
    )
  }

  const performanceScore = report.categories.performance?.score
  const accessibilityScore = report.categories.accessibility?.score

  if (typeof performanceScore !== 'number') {
    throw new Error('categories.performance.score is missing or not a number')
  }
  if (typeof accessibilityScore !== 'number') {
    throw new Error('categories.accessibility.score is missing or not a number')
  }

  writeFileSync(outputPath, `${JSON.stringify(report, null, 2)}\n`, 'utf8')

  return {
    outputPath,
    performanceScore,
    accessibilityScore,
    requestedUrl: report.requestedUrl,
  }
}

async function launchLighthouseContext(
  userDataDir: string,
  port: number,
): Promise<BrowserContext> {
  const launchOptions = {
    headless: true,
    locale: 'en-US',
    viewport: { width: 412, height: 823 },
    args: [
      `--remote-debugging-port=${port}`,
      '--window-size=412,823',
      '--disable-dev-shm-usage',
    ],
  } as const

  const channel = process.env.LH_BROWSER_CHANNEL?.trim()
  if (channel) {
    return chromium.launchPersistentContext(userDataDir, { ...launchOptions, channel })
  }

  try {
    return await chromium.launchPersistentContext(userDataDir, launchOptions)
  } catch (firstError) {
    if (process.env.CI === 'true') throw firstError
    return chromium.launchPersistentContext(userDataDir, { ...launchOptions, channel: 'chrome' })
  }
}

/**
 * Runs Lighthouse against the signed-in global dashboard using a persistent Playwright
 * context (auth + theme survive Lighthouse opening a new page).
 */
export async function runLighthouseDashboard(
  options: LighthouseHarnessOptions,
): Promise<LighthouseHarnessResult> {
  assertLocalhostOrigin(options.pageUrl, options.allowNonLocalhost)

  let token = options.token
  if (!token && !options.requireAuth) {
    ;({ token } = await seedLighthouseDashboardUser(options.theme))
  }
  const authToken = resolveAuthToken({ ...options, token })

  const port = await getFreePort()
  const userDataDir = mkdtempSync(join(tmpdir(), 'lextures-lh-'))

  let context: BrowserContext | undefined
  try {
    context = await launchLighthouseContext(userDataDir, port)

    await registerAuthThemeInitScript(context, authToken, options.theme)

    const page = context.pages()[0] ?? (await context.newPage())
    // Seed auth/theme on the target origin, then wipe IndexedDB before the audited load.
    await page.goto(options.pageUrl, { waitUntil: 'domcontentloaded', timeout: 60_000 })
    await clearIndexedDb(page)
    await page.reload({ waitUntil: 'domcontentloaded', timeout: 60_000 })
    await waitForDashboardReady(page)
    await page.evaluate((theme) => {
      localStorage.setItem('lextures.uiTheme', theme)
      document.documentElement.classList.toggle('dark', theme === 'dark')
      document.documentElement.style.colorScheme = theme === 'dark' ? 'dark' : 'light'
    }, options.theme)
    await assertDocumentTheme(page, options.theme)

    const auditUrl = page.url()
    const result = await lighthouse(
      auditUrl,
      {
        port,
        output: 'json',
        logLevel: 'error',
        disableStorageReset: true,
        onlyCategories: ['performance', 'accessibility'],
      },
      lighthouseConfig(options.theme),
    )

    if (!result?.lhr) {
      throw new Error('Lighthouse did not return a report')
    }

    return validateReport(result.lhr, options.outputPath)
  } finally {
    await context?.close().catch(() => {})
    rmSync(userDataDir, { recursive: true, force: true })
  }
}

export function parseHarnessEnv(): LighthouseHarnessOptions {
  const pageUrl = process.env.PAGE_URL ?? process.env.E2E_BASE_URL ?? 'http://localhost:5173/'
  const themeRaw = (process.env.THEME ?? 'dark').trim().toLowerCase()
  const theme: UiTheme = themeRaw === 'light' ? 'light' : 'dark'
  const outputPath = process.env.LH_OUTPUT ?? DEFAULT_OUTPUT_PATH
  const token = process.env.LH_TOKEN?.trim() || undefined
  const requireAuth = process.env.LH_REQUIRE_AUTH === '1'
  const allowNonLocalhost = process.env.LH_ALLOW_NON_LOCALHOST === '1'

  return {
    pageUrl,
    theme,
    outputPath,
    token,
    requireAuth,
    allowNonLocalhost,
  }
}
