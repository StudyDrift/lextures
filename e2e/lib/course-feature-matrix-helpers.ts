/**
 * Shared Playwright helpers for E2E.1 course feature flag matrix specs.
 */
import type { Page } from '@playwright/test'
import { expect } from '@playwright/test'
import {
  apiGetCourse,
  apiPatchCourseFeatures,
  apiRestoreCourseFeatures,
  apiSnapshotCourseFeatures,
  apiWaitForCourseFeature,
  type CourseFeaturePatch,
} from '../fixtures/api.js'
import {
  type CourseFeatureKey,
  type CourseFeatureMatrixEntry,
  type CourseFeatureNavAssertion,
  readCourseFeatureFlag,
} from './course-feature-matrix.js'

export function featureToggleRow(page: Page, uiLabel: string) {
  return page
    .locator('div')
    .filter({ has: page.locator('p').filter({ hasText: new RegExp(`^${escapeRegExp(uiLabel)}$`) }) })
    .filter({ has: page.getByRole('switch') })
    .last()
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export async function openCourseFeaturesSettings(page: Page, courseCode: string) {
  await page.goto(`/courses/${courseCode}/settings/features`)
  await expect(page.getByRole('heading', { name: /^Course tools$/i })).toBeVisible({
    timeout: 15_000,
  })
}

/**
 * Run a matrix case with snapshot/restore so failed assertions leave the course at baseline.
 */
export async function withCourseFeatureRestore<T>(
  token: string,
  courseCode: string,
  fn: () => Promise<T>,
): Promise<T> {
  const snapshot = await apiSnapshotCourseFeatures(token, courseCode)
  try {
    return await fn()
  } finally {
    await apiRestoreCourseFeatures(token, courseCode, snapshot).catch(() => {})
  }
}

/** Ensure a flag is off before enabling it through the UI. */
export async function ensureCourseFeatureOff(
  token: string,
  courseCode: string,
  entry: Pick<CourseFeatureMatrixEntry, 'key' | 'uiDefaultOn'>,
) {
  await apiPatchCourseFeatures(token, courseCode, { [entry.key]: false } as CourseFeaturePatch)
  await apiWaitForCourseFeature(token, courseCode, entry.key, false, {
    uiDefaultOn: entry.uiDefaultOn,
  })
}

export async function assertUiToggleEnableFlow(
  page: Page,
  token: string,
  courseCode: string,
  entry: CourseFeatureMatrixEntry & { uiLabel: string },
) {
  await ensureCourseFeatureOff(token, courseCode, entry)
  await openCourseFeaturesSettings(page, courseCode)

  const row = featureToggleRow(page, entry.uiLabel)
  const toggle = row.getByRole('switch')
  await expect(toggle, `${courseCode} ${entry.key} switch`).toHaveAttribute('aria-checked', 'false', {
    timeout: 10_000,
  })

  await toggle.click()
  await expect(page.getByRole('status').filter({ hasText: /Saved/i })).toBeVisible({
    timeout: 10_000,
  })
  await expect(toggle).toHaveAttribute('aria-checked', 'true', { timeout: 8_000 })

  await expect
    .poll(
      async () => {
        const data = await apiGetCourse(token, courseCode)
        return readCourseFeatureFlag(data, entry.key, entry.uiDefaultOn)
      },
      { message: `${courseCode} ${entry.key} API after enable`, timeout: 10_000 },
    )
    .toBe(true)

  await page.reload()
  await expect(page.getByRole('heading', { name: /^Course tools$/i })).toBeVisible({
    timeout: 15_000,
  })
  const reloaded = featureToggleRow(page, entry.uiLabel).getByRole('switch')
  await expect(reloaded, `${courseCode} ${entry.key} after reload`).toHaveAttribute(
    'aria-checked',
    'true',
    { timeout: 10_000 },
  )
}

function courseMenu(page: Page) {
  return page.getByRole('navigation', { name: 'Course menu' })
}

function courseSettingsMenu(page: Page) {
  return page.getByRole('navigation', { name: 'Course settings menu' })
}

export async function assertNavGate(
  page: Page,
  token: string,
  courseCode: string,
  key: CourseFeatureKey,
  nav: CourseFeatureNavAssertion,
  uiDefaultOn: boolean,
) {
  const base = `/courses/${courseCode}`
  const routePath = nav.route ? `${base}/${nav.route}` : base

  // Enabled: link present (except pure top-bar cases which use a button).
  await apiPatchCourseFeatures(token, courseCode, { [key]: true } as CourseFeaturePatch)
  await apiWaitForCourseFeature(token, courseCode, key, true, { uiDefaultOn })
  await page.goto(base)
  await expect(courseMenu(page)).toBeVisible({ timeout: 15_000 })

  if (nav.offBehavior === 'top-bar') {
    await expect(page.getByRole('button', { name: nav.linkName })).toBeVisible({ timeout: 10_000 })
  } else if (nav.route.startsWith('settings/')) {
    await page.goto(`${base}/settings/general`)
    await expect(courseSettingsMenu(page).getByRole('link', { name: nav.linkName })).toBeVisible({
      timeout: 10_000,
    })
  } else {
    await expect(courseMenu(page).getByRole('link', { name: nav.linkName })).toBeVisible({
      timeout: 10_000,
    })
  }

  // Disabled: nav hidden and direct route gated.
  await apiPatchCourseFeatures(token, courseCode, { [key]: false } as CourseFeaturePatch)
  await apiWaitForCourseFeature(token, courseCode, key, false, { uiDefaultOn })
  await page.goto(base)
  await expect(courseMenu(page)).toBeVisible({ timeout: 15_000 })

  if (nav.offBehavior === 'top-bar') {
    await expect(page.getByRole('button', { name: nav.linkName })).toHaveCount(0)
    return
  }

  if (nav.route.startsWith('settings/')) {
    await page.goto(`${base}/settings/general`)
    // Sections settings link is conditional; Features tab remains.
    await expect(courseSettingsMenu(page).getByRole('link', { name: nav.linkName })).toHaveCount(0)
  } else {
    await expect(courseMenu(page).getByRole('link', { name: nav.linkName })).toHaveCount(0)
  }

  if (!nav.route) return

  await page.goto(routePath)
  switch (nav.offBehavior) {
    case 'redirect-home':
      await expect(page).not.toHaveURL(new RegExp(`/${escapeRegExp(nav.route)}(?:/|$)`), {
        timeout: 10_000,
      })
      break
    case 'inline-disabled':
    case 'settings-gate':
      await expect(page.getByText(nav.disabledMessage!)).toBeVisible({ timeout: 10_000 })
      break
    case 'nav-only':
      // Page may still render; nav absence is the contract.
      break
    case 'none':
      break
    default: {
      const _exhaustive: never = nav.offBehavior
      throw new Error(`Unhandled offBehavior: ${_exhaustive}`)
    }
  }
}
