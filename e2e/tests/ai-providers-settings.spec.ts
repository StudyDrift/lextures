/**
 * AP.5 / AP.9 — Intelligence admin AI providers UI + test connection (mocked API).
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect } from '@playwright/test'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
const PLACEHOLDER = '••••••••••••'

function uniqueEmail(prefix = 'e2e-ai-providers') {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 10)}@test.invalid`
}

function databaseUrl(): string {
  return (
    process.env.DATABASE_URL ??
    process.env.E2E_DATABASE_URL ??
    'postgres://studydrift:studydrift@localhost:5432/studydrift?sslmode=disable'
  )
}

async function apiSignup(email: string) {
  const res = await fetch(`${API_BASE}/api/v1/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: PASSWORD, display_name: 'E2E AI Admin' }),
  })
  if (!res.ok && res.status !== 409) {
    throw new Error(`signup failed: ${await res.text()}`)
  }
}

async function apiLogin(email: string) {
  const res = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: PASSWORD }),
  })
  if (!res.ok) throw new Error(`login failed: ${await res.text()}`)
  return (await res.json()) as { access_token: string }
}

function bootstrapGlobalAdmin(email: string) {
  execSync(`go run ./cmd/bootstrap-admin -email=${email}`, {
    cwd: path.join(repoRoot, 'server'),
    env: { ...process.env, DATABASE_URL: databaseUrl() },
    stdio: 'pipe',
  })
}

test.describe('Intelligence AI providers (AP.5 / AP.9)', () => {
  test('save Anthropic key, reload shows configured, org test connection toast', async ({ page }) => {
    const email = uniqueEmail()
    await apiSignup(email)
    try {
      bootstrapGlobalAdmin(email)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    const { access_token } = await apiLogin(email)

    let anthropicConfigured = false
    const providers = ['openrouter', 'anthropic', 'openai', 'azure_openai', 'bedrock', 'vertex']

    await page.addInitScript((token) => {
      localStorage.setItem('studydrift_access_token', token)
      localStorage.setItem('lextures-search-shortcut-tip-dismissed', '1')
      localStorage.setItem(
        'lextures.onboarding.v1',
        JSON.stringify({ student: true, teacher: true, admin: true }),
      )
    }, access_token)

    await page.route('**/api/v1/platform/features', async (route) => {
      const res = await route.fetch()
      const data = (await res.json()) as Record<string, unknown>
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...data,
          aiProviderAbstractionEnabled: true,
          aiConfigured: anthropicConfigured,
          openRouterConfigured: anthropicConfigured,
          aiProvidersConfigured: anthropicConfigured ? ['anthropic'] : [],
        }),
      })
    })

    await page.route('**/api/v1/settings/ai/providers', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            credentials: providers.map((p) => ({
              provider: p,
              enabled: true,
              apiKeyConfigured: p === 'anthropic' && anthropicConfigured,
              apiKey: p === 'anthropic' && anthropicConfigured ? PLACEHOLDER : '',
              settings: {},
            })),
            providers,
            tenantByokAllowed: true,
            tenantAllowedProviders: [],
          }),
        })
        return
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ tenantByokAllowed: true, tenantAllowedProviders: [] }),
      })
    })

    await page.route('**/api/v1/settings/ai/providers/**', async (route) => {
      const url = route.request().url()
      const provider = decodeURIComponent(url.split('/').pop() ?? '')
      if (route.request().method() === 'PUT') {
        if (provider === 'anthropic') anthropicConfigured = true
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            provider,
            enabled: true,
            apiKeyConfigured: true,
            apiKey: PLACEHOLDER,
            settings: {},
          }),
        })
        return
      }
      if (route.request().method() === 'DELETE') {
        anthropicConfigured = false
        await route.fulfill({ status: 204 })
        return
      }
      await route.continue()
    })

    await page.route('**/api/v1/settings/ai', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          imageModelId: 'openai/gpt-image-1',
          courseSetupModelId: 'anthropic/claude-3.5-sonnet',
          notebookFlashcardsModelId: 'anthropic/claude-3.5-sonnet',
          vibeActivityModelId: 'anthropic/claude-3.5-sonnet',
          graderAgentModelId: 'anthropic/claude-3.5-sonnet',
          activeProvider: anthropicConfigured ? 'anthropic' : 'openrouter',
          openRouterApiKey: '',
        }),
      })
    })

    await page.route('**/api/v1/settings/ai/models**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          configured: anthropicConfigured,
          provider: anthropicConfigured ? 'anthropic' : 'openrouter',
          models: [
            { id: 'claude-3-5-sonnet', name: 'Claude 3.5 Sonnet' },
            { id: 'claude-3-haiku', name: 'Claude 3 Haiku' },
          ],
        }),
      })
    })

    await page.route('**/api/v1/admin/ai-settings/test', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ok: true,
          provider: 'anthropic',
          authMode: 'api_key',
          latencyMs: 55,
          responsePreview: 'Hello',
        }),
      })
    })

    await page.route('**/api/v1/admin/ai-settings', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            provider: 'anthropic',
            modelAlias: 'claude-3-5-sonnet',
            fallbackProvider: '',
            byokConfigured: true,
            byokApiKey: PLACEHOLDER,
            providers,
            modelAliases: ['claude-3-5-sonnet', 'gpt-4o'],
            credentials: providers.map((p) => ({
              provider: p,
              enabled: true,
              apiKeyConfigured: p === 'anthropic' && anthropicConfigured,
              apiKey: p === 'anthropic' && anthropicConfigured ? PLACEHOLDER : '',
              settings: {},
            })),
          }),
        })
        return
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          provider: 'anthropic',
          modelAlias: 'claude-3-5-sonnet',
          byokConfigured: true,
        }),
      })
    })

    await page.goto('/settings/ai/models')
    await expect(page.getByRole('heading', { name: /^Models$/i })).toBeVisible({ timeout: 20_000 })
    await expect(page.getByRole('heading', { name: /AI providers/i })).toBeVisible()
    await expect(page.getByText('No AI providers are configured yet')).toBeVisible()

    const anthropicCard = page.locator('li').filter({ hasText: 'Anthropic' })
    await anthropicCard.getByLabel(/API key/i).fill('sk-ant-test-key')
    await anthropicCard.getByRole('button', { name: /^Save$/i }).click()
    await expect(page.getByText(/Anthropic saved/i)).toBeVisible({ timeout: 10_000 })

    await page.reload()
    await expect(page.getByRole('heading', { name: /AI providers/i })).toBeVisible({ timeout: 20_000 })
    const anthropicAfterReload = page.locator('li').filter({ hasText: 'Anthropic' })
    await expect(anthropicAfterReload.getByText('Configured')).toBeVisible()
    await expect(anthropicAfterReload.getByText('Default')).toBeVisible()

    // AP.9 FR-3: org admin Test connection smoke (mocked backend).
    await page.goto('/settings/org-branding')
    await expect(page.getByRole('button', { name: /Test connection/i })).toBeVisible({ timeout: 20_000 })
    await page.getByRole('button', { name: /Test connection/i }).click()
    await expect(page.getByText(/Connected via Anthropic/i)).toBeVisible({ timeout: 10_000 })
  })
})
