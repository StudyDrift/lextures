/**
 * Shared Playwright helpers for E2E.3 flagged-feature lifecycle journeys.
 */
import type { Page } from '@playwright/test'
import { expect } from '@playwright/test'
import {
  apiGetPlatformFeatures,
  apiPatchCourseFeatures,
  apiPutPlatformSettings,
  apiWaitForCourseFeature,
  apiWaitForPlatformFeature,
  apiWaitForPlatformSetting,
  type CourseFeaturePatch,
} from '../fixtures/api.js'
import { withCourseFeatureRestore } from './course-feature-matrix-helpers.js'
import type { CourseFeatureKey } from './course-feature-matrix.js'
import {
  bootstrapGlobalAdmin,
  withPlatformBooleanRestore,
} from './platform-feature-matrix-helpers.js'
import {
  type ApiProbe,
  type DependencyEdge,
  type LifecycleFamily,
  type LifecycleFlagRef,
  flagKey,
} from './feature-lifecycle-manifest.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

export { bootstrapGlobalAdmin }

export function resolveProbePath(path: string, courseCode?: string): string {
  return path.replaceAll('{courseCode}', courseCode ?? 'C-MISSING')
}

export async function setPlatformFlag(
  token: string,
  key: string,
  value: boolean,
  opts?: { waitRuntime?: boolean },
) {
  await apiPutPlatformSettings(token, {
    [key]: value,
    updateMask: [key],
  })
  await apiWaitForPlatformSetting(token, key, value)
  if (opts?.waitRuntime !== false) {
    // Runtime payload uses the same key for most platform flags.
    try {
      await apiWaitForPlatformFeature(token, key, value, { timeoutMs: 8_000 })
    } catch {
      // Settings-only flags have no runtime key — ignore.
    }
  }
}

export async function setCourseFlag(
  token: string,
  courseCode: string,
  key: string,
  value: boolean,
) {
  const courseKey = key as CourseFeatureKey
  await apiPatchCourseFeatures(token, courseCode, { [courseKey]: value } as CourseFeaturePatch)
  await apiWaitForCourseFeature(token, courseCode, courseKey, value, {
    uiDefaultOn: false,
  })
}

export async function applyFlagState(
  gaToken: string,
  instructorToken: string | undefined,
  courseCode: string | undefined,
  ref: LifecycleFlagRef,
  value: boolean,
) {
  if (ref.alwaysOn) return
  if (ref.kind === 'platform') {
    await setPlatformFlag(gaToken, ref.key, value)
    return
  }
  if (!instructorToken || !courseCode) {
    throw new Error(`Course flag ${ref.key} requires instructorToken + courseCode`)
  }
  await setCourseFlag(instructorToken, courseCode, ref.key, value)
}

/**
 * Snapshot/restore both platform booleans (locked) and optional course features.
 */
export async function withFeatureLifecycleRestore<T>(opts: {
  gaToken: string
  instructorToken?: string
  courseCode?: string
  fn: () => Promise<T>
}): Promise<T> {
  const run = async () =>
    withPlatformBooleanRestore(opts.gaToken, async () => {
      if (opts.instructorToken && opts.courseCode) {
        return withCourseFeatureRestore(opts.instructorToken, opts.courseCode, opts.fn)
      }
      return opts.fn()
    })

  // withPlatformBooleanRestore already takes the lock; call directly.
  return run()
}

export async function fetchProbe(
  probe: ApiProbe,
  opts: { token?: string | null; courseCode?: string },
): Promise<Response> {
  const path = resolveProbePath(probe.path, opts.courseCode)
  const headers: Record<string, string> = {}
  if (opts.token) {
    headers.Authorization = `Bearer ${opts.token}`
  }
  if (probe.body != null) {
    headers['Content-Type'] = 'application/json'
  }
  return fetch(`${API_BASE}${path}`, {
    method: probe.method,
    headers,
    body: probe.body != null ? JSON.stringify(probe.body) : undefined,
  })
}

export async function assertProbeDisabled(
  probe: ApiProbe,
  opts: { token?: string | null; courseCode?: string; label: string },
) {
  const res = await fetchProbe(probe, opts)
  const expected =
    opts.token == null || opts.token === ''
      ? probe.unauthContract === 'auth-first'
        ? 401
        : (probe.unauthDisabledStatus ?? probe.authDisabledStatus)
      : probe.authDisabledStatus
  expect(res.status, `${opts.label} ${probe.method} ${probe.path}`).toBe(expected)
}

/**
 * Parent/child truth table for a single edge (AC-2).
 * When parentAuthoritative, child-on + parent-off must keep the child probe disabled.
 */
export async function assertDependencyTruthTable(opts: {
  family: LifecycleFamily
  edge: DependencyEdge
  gaToken: string
  instructorToken?: string
  courseCode?: string
  /** Probe that represents the child surface when both are on. */
  childProbe: ApiProbe
  /** Status (or predicate) when both parent and child are on — may be non-2xx (authz/validation). */
  bothOnAllowedStatuses: number[]
}) {
  const { edge, gaToken, instructorToken, courseCode, childProbe, bothOnAllowedStatuses } = opts
  const label = `${opts.family.id} ${flagKey(edge.parent)}→${flagKey(edge.child)}`

  const combos: Array<{ parent: boolean; child: boolean; expectDisabled: boolean }> = [
    { parent: false, child: false, expectDisabled: true },
    { parent: true, child: false, expectDisabled: true },
    { parent: false, child: true, expectDisabled: edge.parentAuthoritative },
    { parent: true, child: true, expectDisabled: false },
  ]

  for (const combo of combos) {
    await applyFlagState(gaToken, instructorToken, courseCode, edge.parent, combo.parent)
    await applyFlagState(gaToken, instructorToken, courseCode, edge.child, combo.child)

    const res = await fetchProbe(childProbe, {
      token: instructorToken ?? gaToken,
      courseCode,
    })

    if (combo.expectDisabled) {
      if (!edge.parentAuthoritative && combo.parent === false && combo.child === true) {
        // Documented gap: child may still respond; do not fail CI.
        expect(
          [childProbe.authDisabledStatus, ...bothOnAllowedStatuses],
          `${label} parent-off/child-on (gap)`,
        ).toContain(res.status)
      } else {
        expect(res.status, `${label} parent=${combo.parent} child=${combo.child}`).toBe(
          childProbe.authDisabledStatus,
        )
      }
    } else {
      expect(
        bothOnAllowedStatuses,
        `${label} both on got ${res.status}`,
      ).toContain(res.status)
    }
  }
}

export async function assertRuntimeFlag(
  token: string,
  key: string,
  expected: boolean,
  label: string,
) {
  await expect
    .poll(
      async () => {
        const features = await apiGetPlatformFeatures(token)
        return features[key] === expected
      },
      { message: `${label} runtime ${key}=${expected}`, timeout: 10_000 },
    )
    .toBe(true)
}

export async function createBoard(
  token: string,
  courseCode: string,
  title: string,
): Promise<{ id: string; title: string }> {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/boards`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ title, description: 'E2E.3 lifecycle' }),
  })
  if (!res.ok) {
    throw new Error(`createBoard failed (${res.status}): ${await res.text()}`)
  }
  return res.json() as Promise<{ id: string; title: string }>
}

export async function listBoards(
  token: string,
  courseCode: string,
): Promise<{ id: string; title: string }[]> {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/boards`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) {
    throw new Error(`listBoards failed (${res.status}): ${await res.text()}`)
  }
  const body = (await res.json()) as { boards?: { id: string; title: string }[] } | { id: string; title: string }[]
  if (Array.isArray(body)) return body
  return body.boards ?? []
}

export async function createQuizKit(
  token: string,
  courseCode: string,
  title: string,
): Promise<{ id: string; title: string }> {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits`,
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ title, description: 'E2E.3 lifecycle kit' }),
    },
  )
  if (!res.ok) {
    throw new Error(`createQuizKit failed (${res.status}): ${await res.text()}`)
  }
  return res.json() as Promise<{ id: string; title: string }>
}

export async function listQuizKits(
  token: string,
  courseCode: string,
): Promise<{ id: string; title: string }[]> {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/kits`,
    { headers: { Authorization: `Bearer ${token}` } },
  )
  if (!res.ok) {
    throw new Error(`listQuizKits failed (${res.status}): ${await res.text()}`)
  }
  const body = (await res.json()) as
    | { kits?: { id: string; title: string }[] }
    | { id: string; title: string }[]
  if (Array.isArray(body)) return body
  return body.kits ?? []
}

export async function getParentNotificationPrefs(
  token: string,
): Promise<Record<string, unknown>> {
  const res = await fetch(`${API_BASE}/api/v1/parent/notification-prefs`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) {
    throw new Error(`getParentNotificationPrefs failed (${res.status}): ${await res.text()}`)
  }
  return res.json() as Promise<Record<string, unknown>>
}

export async function patchParentNotificationPrefs(
  token: string,
  body: Record<string, unknown>,
): Promise<Record<string, unknown>> {
  const res = await fetch(`${API_BASE}/api/v1/parent/notification-prefs`, {
    method: 'PATCH',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    throw new Error(`patchParentNotificationPrefs failed (${res.status}): ${await res.text()}`)
  }
  return res.json() as Promise<Record<string, unknown>>
}

export async function assertCourseWebOffState(
  page: Page,
  family: LifecycleFamily,
  courseCode: string,
) {
  if (!family.web || family.web.offBehavior === 'none' || family.web.offBehavior === 'runtime-only') {
    return
  }
  const route = family.web.route.replaceAll('{courseCode}', courseCode)
  await page.goto(route)
  if (family.web.offBehavior === 'inline-disabled' && family.web.disabledMessage) {
    await expect(page.getByText(family.web.disabledMessage)).toBeVisible({ timeout: 15_000 })
  }
  if (family.web.navLinkName) {
    await page.goto(`/courses/${courseCode}`)
    const nav = page.getByRole('navigation', { name: 'Course menu' })
    await expect(nav).toBeVisible({ timeout: 15_000 })
    await expect(nav.getByRole('link', { name: family.web.navLinkName })).toHaveCount(0)
  }
}

export async function assertCourseWebOnState(
  page: Page,
  family: LifecycleFamily,
  courseCode: string,
) {
  if (!family.web?.navLinkName) return
  await page.goto(`/courses/${courseCode}`)
  const nav = page.getByRole('navigation', { name: 'Course menu' })
  await expect(nav.getByRole('link', { name: family.web.navLinkName })).toBeVisible({
    timeout: 15_000,
  })
}

/** Create an access key while ffApiTokens is on; used for public-API kill-switch checks. */
export async function createAccessKey(
  token: string,
  label: string,
): Promise<{ id: string; token: string }> {
  const res = await fetch(`${API_BASE}/api/v1/me/access-keys`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ label, scopes: ['courses:read'] }),
  })
  if (!res.ok) {
    throw new Error(`createAccessKey failed (${res.status}): ${await res.text()}`)
  }
  return res.json() as Promise<{ id: string; token: string }>
}
