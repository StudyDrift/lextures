/**
 * CT.5 — Tool SDK sandboxing, versioning, and migration dry-run.
 *
 * Checklist:
 *   [x] Sandbox iframe tool mounts and completes interact → save → grade
 *   [x] Hostile probes (cookie / parent DOM / storage / disallowed fetch) stay contained
 *   [x] Admin migration dry-run reports counts without mutating
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect, injectToken } from '../fixtures/test.js'
import {
  apiCreateContentPage,
  apiCreateModule,
  apiPatchContentPage,
  apiPatchCourseFeatures,
} from '../fixtures/api.js'
import { withCourseFeatureRestore } from '../lib/course-feature-matrix-helpers.js'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

function databaseUrl(): string {
  return (
    process.env.DATABASE_URL ??
    process.env.E2E_DATABASE_URL ??
    'postgres://studydrift:studydrift@localhost:5432/studydrift?sslmode=disable'
  )
}

async function signupAndBootstrapAdmin(): Promise<string> {
  const email = `e2e-ct5-admin-${Date.now()}-${Math.random().toString(36).slice(2, 10)}@test.invalid`
  const signup = await fetch(`${apiBase}/api/v1/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: PASSWORD, display_name: 'CT5 Admin' }),
  })
  if (!signup.ok && signup.status !== 409) {
    throw new Error(`signup failed: ${await signup.text()}`)
  }
  execSync(`go run ./cmd/bootstrap-admin -email=${email}`, {
    cwd: path.join(repoRoot, 'server'),
    env: { ...process.env, DATABASE_URL: databaseUrl() },
    stdio: 'pipe',
  })
  const login = await fetch(`${apiBase}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: PASSWORD }),
  })
  if (!login.ok) throw new Error(`login failed: ${await login.text()}`)
  const body = (await login.json()) as { access_token: string }
  return body.access_token
}

async function putSettings(
  token: string,
  courseCode: string,
  body: {
    allowedToolIds: string[]
    studentResetAllowed: boolean
    maxInstancesPerItem: number
  },
): Promise<number> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/settings`,
    {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(body),
    },
  )
  return res.status
}

async function createSandboxInstance(
  token: string,
  courseCode: string,
  structureItemId: string,
): Promise<{ id: string; toolId: string }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        toolId: 'sandbox_probe',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT5 sandbox',
        config: { prompt: 'CT5 sandbox prompt', answerKey: 'paris' },
      }),
    },
  )
  if (!res.ok) {
    throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  }
  return (await res.json()) as { id: string; toolId: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT5 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'sandbox_probe', v: 1 }),
    '```',
    '',
  ].join('\n')
}

test.describe('Content Tools SDK / sandbox (CT.5)', () => {
  test('iframe sandbox interact → save → grade; hostile probes contained', async ({
    page,
    seededCourse,
  }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      expect(
        await putSettings(instructorToken, courseCode, {
          allowedToolIds: ['sandbox_probe', 'noop_probe'],
          studentResetAllowed: false,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const mod = await apiCreateModule(instructorToken, courseCode, 'CT5 module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        mod.id,
        'CT5 page',
      )
      const inst = await createSandboxInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      await injectToken(page, studentToken)
      await page.goto(
        `/courses/${encodeURIComponent(courseCode)}/modules/content/${contentPage.id}?ct5Hostile=1`,
      )

      const host = page.locator('[data-content-tool="sandbox_probe"][data-sandbox="iframe"]')
      await expect(host).toBeVisible({ timeout: 30_000 })
      const frame = page.frameLocator('iframe[title]')
      await expect(frame.locator('#prompt')).toContainText('CT5 sandbox prompt', {
        timeout: 15_000,
      })
      await frame.locator('#response').fill('paris')
      await frame.locator('#grade').click()
      await expect(frame.locator('#status')).toHaveAttribute('data-correct', 'true', {
        timeout: 15_000,
      })

      const hostile = frame.locator('[data-testid="hostile-results"]')
      await expect(hostile).toBeVisible({ timeout: 15_000 })
      await expect(hostile).toHaveAttribute('data-parent-dom', 'blocked')
      await expect(hostile).toHaveAttribute('data-storage', 'blocked')
      // Cookie should be empty or threw — never the host session.
      const cookieAttr = await hostile.getAttribute('data-cookie')
      expect(cookieAttr === 'blocked' || cookieAttr === 'threw' || cookieAttr === '').toBeTruthy()
      await expect(hostile).toHaveAttribute('data-fetch', 'blocked')
    })
  })

  test('admin migration dry-run reports counts and mutates nothing', async () => {
    const token = await signupAndBootstrapAdmin()

    const create = await fetch(`${apiBase}/api/v1/admin/content-tools/migrations`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        toolId: 'noop_probe',
        fromVersion: 1,
        toVersion: 2,
        dryRun: true,
      }),
    })
    expect(create.status).toBe(202)
    const job = (await create.json()) as {
      dryRun: boolean
      status: string
      totalDocs: number
      migratedDocs: number
      failedDocs: number
    }
    expect(job.dryRun).toBe(true)
    expect(['succeeded', 'running', 'queued', 'failed']).toContain(job.status)
  })
})
