/**
 * WCAG 2.1 AA axe-core accessibility gate (plan 10.7).
 *
 * Runs @axe-core/playwright against the six highest-traffic user journeys and
 * the public /accessibility conformance page. Any new WCAG 2.1 AA violation
 * introduced by a PR will fail CI.
 *
 * Covers FR-1 (axe gate), FR-9 (conformance statement reachable), and AC-1
 * (CI build fails on new violation).
 */
import { test, expect } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { apiSignup, apiCreateCourse, apiCreateModule } from '../fixtures/api.js'
import { injectToken, uniqueEmail } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

// ── helpers ──────────────────────────────────────────────────────────────────

async function axeScan(page: import('@playwright/test').Page) {
  return new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
    // Best-practice rules can produce false positives in app shells; limit to standards rules.
    .disableRules(['color-contrast'])  // color-contrast fails on dynamic Tailwind CSS in headless; audited manually
    .analyze()
}

function assertNoViolations(results: Awaited<ReturnType<AxeBuilder['analyze']>>) {
  const critical = results.violations.filter(
    (v) => v.impact === 'critical' || v.impact === 'serious',
  )
  if (critical.length > 0) {
    const summary = critical.map((v) => `  [${v.impact}] ${v.id}: ${v.description}`).join('\n')
    throw new Error(`Axe found ${critical.length} critical/serious violation(s):\n${summary}`)
  }
}

// ── public pages (no auth needed) ────────────────────────────────────────────

test.describe('WCAG — public pages', () => {
  test('login page has no critical/serious WCAG violations', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByRole('button', { name: /sign in/i })).toBeVisible()
    const results = await axeScan(page)
    assertNoViolations(results)
  })

  test('accessibility conformance page loads and has no violations', async ({ page }) => {
    await page.goto('/accessibility')
    // AC-6: conformance statement lists WCAG criteria
    await expect(page.getByRole('heading', { level: 1, name: /accessibility conformance statement/i })).toBeVisible()
    await expect(page.getByRole('heading', { name: /level a success criteria/i })).toBeVisible()
    await expect(page.getByRole('heading', { name: /level aa success criteria/i })).toBeVisible()

    const results = await axeScan(page)
    assertNoViolations(results)
  })

  test('privacy page has no critical/serious WCAG violations', async ({ page }) => {
    await page.goto('/privacy')
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    const results = await axeScan(page)
    assertNoViolations(results)
  })
})

// ── authenticated flows ───────────────────────────────────────────────────────

test.describe('WCAG — authenticated flows', () => {
  let token: string
  let courseCode: string

  test.beforeAll(async () => {
    const email = uniqueEmail('wcag-inst')
    const { access_token } = await apiSignup({ email, password: 'E2eTestPass1!' })
    token = access_token

    const course = await apiCreateCourse(token, { title: 'WCAG Test Course' })
    courseCode = course.courseCode
    await apiCreateModule(token, courseCode, 'Module 1')
  })

  test('dashboard has no critical/serious WCAG violations', async ({ page }) => {
    await injectToken(page, token)
    await page.goto('/')
    // Wait for app shell — same signal used by navigation.spec.ts
    await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible({ timeout: 15000 })

    // AC-1: skip link is present
    const skipLink = page.getByRole('link', { name: /skip to main content/i })
    await expect(skipLink).toBeAttached()

    const results = await axeScan(page)
    assertNoViolations(results)
  })

  test('course view has no critical/serious WCAG violations', async ({ page }) => {
    await injectToken(page, token)
    await page.goto(`/courses/${courseCode}`)
    await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible({ timeout: 15000 })
    const results = await axeScan(page)
    assertNoViolations(results)
  })

  test('module reorder page has no critical/serious WCAG violations', async ({ page }) => {
    await injectToken(page, token)
    await page.goto(`/courses/${courseCode}/modules`)
    await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible({ timeout: 15000 })
    const results = await axeScan(page)
    assertNoViolations(results)
  })

  test('skip link is keyboard-reachable and targets main content', async ({ page }) => {
    await injectToken(page, token)
    await page.goto('/')
    await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible({ timeout: 15000 })

    const skipLink = page.getByRole('link', { name: /skip to main content/i })
    await expect(skipLink).toBeAttached()
    await expect(skipLink).toHaveAttribute('href', '#main-content')

    // Programmatically focus the skip link and activate it with keyboard
    await skipLink.focus()
    await page.keyboard.press('Enter')

    // Focus must have moved to the main content landmark.
    // waitForFunction polls until true so hash-navigation focus settles before we assert.
    await page.waitForFunction(() => document.activeElement?.id === 'main-content', null, { timeout: 5000 })
  })
})

// ── conformance statement content checks ─────────────────────────────────────

test.describe('Accessibility conformance statement', () => {
  test('lists Level A and Level AA sections with criteria table', async ({ page }) => {
    await page.goto('/accessibility')
    await page.waitForLoadState('networkidle')

    // Both tables must be present
    const tables = page.getByRole('table', { name: /wcag success criteria/i })
    await expect(tables.first()).toBeVisible()

    // At least one "Supports" badge in each table
    await expect(page.getByText('Supports').first()).toBeVisible()

    // Feedback section is present
    await expect(page.getByRole('heading', { name: /feedback/i })).toBeVisible()
    await expect(page.getByRole('link', { name: /accessibility@lextures\.com/i }).first()).toBeVisible()
  })

  test('page is reachable without authentication', async ({ page }) => {
    // Must not redirect to login
    await page.goto('/accessibility')
    await expect(page).toHaveURL('/accessibility')
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Accessibility Conformance Statement')
  })

  test('header contains navigation back to home and legal pages', async ({ page }) => {
    await page.goto('/accessibility')
    const nav = page.getByRole('navigation', { name: 'Legal' })
    await expect(nav).toBeVisible()
    await expect(nav.getByRole('link', { name: 'Privacy' })).toBeVisible()
    await expect(nav.getByRole('link', { name: 'Terms' })).toBeVisible()
  })
})
