const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

const E2E_ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
const E2E_ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

/** Enable common platform features via Settings → Global platform (replaces FEATURE_* env vars). */
export async function seedE2EPlatformFeatures(): Promise<void> {
  await fetch(`${apiBase}/api/v1/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: E2E_ADMIN_EMAIL,
      password: E2E_ADMIN_PASSWORD,
      display_name: 'E2E Admin',
    }),
  }).catch(() => {})

  const loginRes = await fetch(`${apiBase}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: E2E_ADMIN_EMAIL, password: E2E_ADMIN_PASSWORD }),
  })
  if (!loginRes.ok) {
    return
  }
  const { access_token: token } = (await loginRes.json()) as { access_token: string }

  const body = {
    h5pEnabled: true,
    oerLibraryEnabled: true,
    oerStub: true,
    studentProgressEnabled: true,
    atRiskAlertsEnabled: true,
    equationEditorEnabled: true,
    updateMask: [
      'h5pEnabled',
      'oerLibraryEnabled',
      'oerStub',
      'studentProgressEnabled',
      'atRiskAlertsEnabled',
      'equationEditorEnabled',
    ],
  }
  await fetch(`${apiBase}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(body),
  })
}
