const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

const E2E_ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
const E2E_ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

/** Enable common platform features via Settings → Global platform (replaces FEATURE_* env vars). */
export async function seedE2EPlatformFeatures(): Promise<void> {
  const signupRes = await fetch(`${apiBase}/api/v1/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: E2E_ADMIN_EMAIL,
      password: E2E_ADMIN_PASSWORD,
      display_name: 'E2E Admin',
    }),
  })
  if (!signupRes.ok && signupRes.status !== 409) {
    const body = await signupRes.text()
    throw new Error(`E2E admin signup failed (${signupRes.status}): ${body}`)
  }

  const loginRes = await fetch(`${apiBase}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: E2E_ADMIN_EMAIL, password: E2E_ADMIN_PASSWORD }),
  })
  if (!loginRes.ok) {
    const body = await loginRes.text()
    throw new Error(`E2E admin login failed (${loginRes.status}): ${body}`)
  }
  const { access_token: token } = (await loginRes.json()) as { access_token: string }

  const putRes = await fetch(`${apiBase}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      h5pEnabled: true,
      oerLibraryEnabled: true,
      oerStub: true,
      studentProgressEnabled: true,
      selfReflectionEnabled: true,
      outcomesReportEnabled: true,
      atRiskAlertsEnabled: true,
      equationEditorEnabled: true,
      reportExportEnabled: true,
      xapiEmissionEnabled: true,
      instructorInsightsEnabled: true,
      updateMask: [
        'h5pEnabled',
        'oerLibraryEnabled',
        'oerStub',
        'studentProgressEnabled',
        'selfReflectionEnabled',
        'outcomesReportEnabled',
        'atRiskAlertsEnabled',
        'equationEditorEnabled',
        'reportExportEnabled',
        'xapiEmissionEnabled',
        'instructorInsightsEnabled',
      ],
    }),
  })
  if (!putRes.ok) {
    const body = await putRes.text()
    throw new Error(`E2E platform settings seed failed (${putRes.status}): ${body}`)
  }
}

/** Enable engagement tracking for specs that post heartbeat events (not enabled in global seed). */
export async function enableEngagementTrackingForE2E(): Promise<void> {
  const loginRes = await fetch(`${apiBase}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: E2E_ADMIN_EMAIL, password: E2E_ADMIN_PASSWORD }),
  })
  if (!loginRes.ok) {
    const body = await loginRes.text()
    throw new Error(`E2E admin login failed (${loginRes.status}): ${body}`)
  }
  const { access_token: token } = (await loginRes.json()) as { access_token: string }
  const putRes = await fetch(`${apiBase}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      engagementTrackingEnabled: true,
      updateMask: ['engagementTrackingEnabled'],
    }),
  })
  if (!putRes.ok) {
    const body = await putRes.text()
    throw new Error(`E2E engagement enable failed (${putRes.status}): ${body}`)
  }
}
