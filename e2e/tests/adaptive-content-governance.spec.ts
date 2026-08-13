/**
 * AC.8 — Governance: contests, oversight, kill-switch, quarantine API.
 *
 * Checklist:
 *   [x] Student can open a contest on a unit
 *   [x] Instructor lists and resolves contests
 *   [x] Admin oversight endpoint returns summary
 *   [x] Admin quarantine + kill-switch endpoints work
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect, uniqueEmail } from '../fixtures/test.js'
import {
  apiCreateContentPage,
  apiLogin,
  apiPatchCourseFeatures,
  apiSignup,
} from '../fixtures/api.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const ADMIN_PASSWORD = 'E2eTestPass1!LongRandomGovernance'

async function apiCreateAdaptiveUnit(
  token: string,
  courseCode: string,
  body: Record<string, unknown>,
): Promise<{ id: string }> {
  const deadline = Date.now() + 15_000
  for (;;) {
    const res = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/adaptive-content/units`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(body),
      },
    )
    if (res.ok) {
      return res.json() as Promise<{ id: string }>
    }
    const text = await res.text()
    const killSwitchBusy = res.status === 503 && text.includes('kill-switch engaged')
    if (!killSwitchBusy || Date.now() >= deadline) {
      throw new Error(`Create ACE unit failed (${res.status}): ${text}`)
    }
    await new Promise((r) => setTimeout(r, 400))
  }
}

function bootstrapGlobalAdmin(email: string) {
  execSync(`go run ./cmd/bootstrap-admin -email=${email}`, {
    cwd: path.join(repoRoot, 'server'),
    stdio: 'pipe',
    env: process.env,
  })
}

test.describe('Adaptive content governance (AC.8)', () => {
  test('API: contest, resolve, oversight, quarantine, kill-switch', async ({
    seededCourse,
  }) => {
    await apiPatchCourseFeatures(seededCourse.instructorToken, seededCourse.courseCode, {
      adaptiveContentEnabled: true,
    })

    const page = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'ACE Governance Content',
    )
    const unit = await apiCreateAdaptiveUnit(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      {
        targetKind: 'module',
        targetModuleItemId: seededCourse.moduleId,
        baseContentItemId: page.id,
        status: 'active',
        triggerMode: 'mastery_snapshot',
      },
    )

    const contestRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/adaptive-content/units/${encodeURIComponent(unit.id)}/contest`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${seededCourse.studentToken}`,
        },
        body: JSON.stringify({ reason: 'This adaptation seems wrong' }),
      },
    )
    expect(contestRes.status).toBe(201)
    const contest = (await contestRes.json()) as { id: string; status: string }
    expect(contest.status).toBe('open')

    const listRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/adaptive-content/contests?status=open`,
      { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
    )
    expect(listRes.ok).toBeTruthy()
    const list = (await listRes.json()) as { contests: Array<{ id: string }> }
    expect(list.contests.some((c) => c.id === contest.id)).toBeTruthy()

    const resolveRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/adaptive-content/contests/${encodeURIComponent(contest.id)}/resolve`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${seededCourse.instructorToken}`,
        },
        body: JSON.stringify({ status: 'resolved' }),
      },
    )
    expect(resolveRes.ok).toBeTruthy()

    const adminEmail = uniqueEmail('ace-admin')
    await apiSignup({
      email: adminEmail,
      password: ADMIN_PASSWORD,
      displayName: 'ACE Admin',
    })
    try {
      bootstrapGlobalAdmin(adminEmail)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    const { access_token: adminToken } = await apiLogin({
      email: adminEmail,
      password: ADMIN_PASSWORD,
    })

    const oversightRes = await fetch(`${apiBase}/api/v1/admin/adaptive-content/oversight`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    })
    expect(oversightRes.ok).toBeTruthy()
    const oversight = (await oversightRes.json()) as {
      killSwitch: boolean
      openContests: number
      dpiaDocPath: string
    }
    expect(typeof oversight.killSwitch).toBe('boolean')
    expect(oversight.dpiaDocPath).toContain('ace-dpia')

    const quarantineRes = await fetch(`${apiBase}/api/v1/admin/adaptive-content/quarantine`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${adminToken}`,
      },
      body: JSON.stringify({ unitId: unit.id, reason: 'e2e incident drill' }),
    })
    expect(quarantineRes.ok).toBeTruthy()

    try {
      const killRes = await fetch(`${apiBase}/api/v1/admin/adaptive-content/kill-switch`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${adminToken}`,
        },
        body: JSON.stringify({ engage: true }),
      })
      expect(killRes.ok).toBeTruthy()
    } finally {
      // Always disengage: CI shards share one API and fullyParallel workers.
      const off = await fetch(`${apiBase}/api/v1/admin/adaptive-content/kill-switch`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${adminToken}`,
        },
        body: JSON.stringify({ engage: false }),
      })
      expect(off.ok).toBeTruthy()
    }
  })
})
