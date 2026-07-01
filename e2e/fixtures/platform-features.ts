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
      scormIngestionEnabled: true,
      oerLibraryEnabled: true,
      oerStub: true,
      studentProgressEnabled: true,
      selfReflectionEnabled: true,
      outcomesReportEnabled: true,
      atRiskAlertsEnabled: true,
      equationEditorEnabled: true,
      readingLevelEnabled: true,
      graderAgentEnabled: true,
      graderAgentReviewInboxEnabled: true,
      graderAgentSuggestModeEnabled: true,
      graderAgentTextEntryGradingEnabled: true,
      graderAgentVisionGradingEnabled: false,
      graderAgentRunFiltersEnabled: true,
      graderAgentCancelRunEnabled: true,
      altTextEnforcementEnabled: true,
      ffAltTextEnforcement: true,
      speechToTextEnabled: true,
      readAloudEnabled: true,
      ffReadAloud: true,
      translationMemoryEnabled: true,
      reportExportEnabled: true,
      xapiEmissionEnabled: true,
      instructorInsightsEnabled: true,
      coppaWorkflowEnabled: true,
      isoIsmsEnabled: true,
      securityDisclosureModuleEnabled: true,
      engagementTrackingEnabled: true,
      adminAuditLogEnabled: true,
      avScanningEnabled: true,
      clamavStub: true,
      storageQuotasEnabled: true,
      dataResidencyEnabled: true,
      itemAnalysisEnabled: true,
      videoCaptionsEnabled: true,
      autoCaptioningEnabled: true,
      // Flags migrated off env (now DB-managed via this seed).
      rtlEnabled: true,
      gdprModuleEnabled: true,
      ccpaModuleEnabled: true,
      statePrivacyEnabled: true,
      backupModuleEnabled: true,
      ferpaWorkflowEnabled: true,
      ffSisIntegration: true,
      ffCatalogIntegration: true,
      ffEnrollmentStateMachine: true,
      ffIncompleteGradeWorkflow: true,
      ffWhatifGrades: true,
      ffBookstoreIntegration: true,
      ffLibrary: true,
      ffLibraryIntegration: true,
      ffBroadcasts: true,
      ffClassroomSignals: true,
      ffPlagiarismChecks: true,
      originalityDetectionEnabled: true,
      originalityStubExternal: true,
      ffAiStudyBuddy: true,
      ffPersistentTutor: true,
      ffWebhooks: true,
      ffBotSlack: true,
      ffBotDiscord: true,
      ffBotTeams: true,
      updateMask: [
        'h5pEnabled',
        'scormIngestionEnabled',
        'oerLibraryEnabled',
        'oerStub',
        'studentProgressEnabled',
        'selfReflectionEnabled',
        'outcomesReportEnabled',
        'atRiskAlertsEnabled',
        'equationEditorEnabled',
        'readingLevelEnabled',
        'graderAgentEnabled',
        'graderAgentReviewInboxEnabled',
        'graderAgentSuggestModeEnabled',
        'graderAgentTextEntryGradingEnabled',
        'graderAgentRunFiltersEnabled',
        'graderAgentCancelRunEnabled',
        'altTextEnforcementEnabled',
        'ffAltTextEnforcement',
        'speechToTextEnabled',
        'readAloudEnabled',
        'ffReadAloud',
        'translationMemoryEnabled',
        'reportExportEnabled',
        'xapiEmissionEnabled',
        'instructorInsightsEnabled',
        'coppaWorkflowEnabled',
        'isoIsmsEnabled',
        'securityDisclosureModuleEnabled',
        'engagementTrackingEnabled',
        'adminAuditLogEnabled',
        'avScanningEnabled',
        'clamavStub',
        'storageQuotasEnabled',
        'dataResidencyEnabled',
        'itemAnalysisEnabled',
        'videoCaptionsEnabled',
        'autoCaptioningEnabled',
        'rtlEnabled',
        'gdprModuleEnabled',
        'ccpaModuleEnabled',
        'statePrivacyEnabled',
        'backupModuleEnabled',
        'ferpaWorkflowEnabled',
        'ffSisIntegration',
        'ffCatalogIntegration',
        'ffEnrollmentStateMachine',
        'ffIncompleteGradeWorkflow',
        'ffWhatifGrades',
        'ffBookstoreIntegration',
        'ffLibrary',
        'ffLibraryIntegration',
        'ffBroadcasts',
        'ffClassroomSignals',
        'ffPlagiarismChecks',
        'originalityDetectionEnabled',
        'originalityStubExternal',
        'ffAiStudyBuddy',
        'ffPersistentTutor',
        'ffWebhooks',
        'ffBotSlack',
        'ffBotDiscord',
        'ffBotTeams',
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

// --- Runtime feature probes (reliable across env/platform/seed; used to replace legacy FEATURE_* checks) ---

async function probeEnabled(path: string): Promise<boolean> {
  const res = await fetch(`${apiBase}${path}`)
  // 401/403 means feature enabled (auth reached), 404 means feature guard returned not-found first.
  return res.status === 401 || res.status === 403
}

export async function isH5PEnabled(): Promise<boolean> {
  if (process.env.FEATURE_H5P === 'true') return true
  // unauthed to guarded h5p route: 401 when on, 404 when off
  const res = await fetch(`${apiBase}/api/v1/courses/FAKE/h5p/00000000-0000-0000-0000-000000000000`)
  return res.status === 401
}

export async function isScormIngestionEnabled(): Promise<boolean> {
  if (process.env.FEATURE_SCORM_INGESTION === 'true') return true
  const res = await fetch(
    `${apiBase}/api/v1/courses/FAKE/scorm-items/00000000-0000-0000-0000-000000000000`,
  )
  return res.status === 401
}

export async function isOEREnabled(): Promise<boolean> {
  if (process.env.FEATURE_OER_LIBRARY === 'true') return true
  // oer search may be public-ish; probe a course-scoped oer add attempt or use settings
  const res = await fetch(`${apiBase}/api/v1/courses/FAKE/oer`)
  return res.status === 401 || res.status === 403
}

export async function isEngagementEnabled(): Promise<boolean> {
  if (process.env.FEATURE_ENGAGEMENT_TRACKING === 'true') return true
  // POST only — GET returns 405 before the feature gate is evaluated.
  const res = await fetch(`${apiBase}/api/v1/analytics/events`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify([]),
  })
  return res.status === 401
}

export async function isAtRiskEnabled(): Promise<boolean> {
  if (process.env.FEATURE_AT_RISK_ALERTS === 'true') return true
  const res = await fetch(`${apiBase}/api/v1/courses/FAKE/at-risk`)
  return res.status === 401 || res.status === 403
}

export async function isStorageQuotasEnabled(): Promise<boolean> {
  if (process.env.STORAGE_QUOTAS_ENABLED === 'true') return true
  const res = await fetch(`${apiBase}/api/v1/courses/FAKE/storage-usage`)
  return res.status === 401 || res.status === 403
}

export async function isAVEnabled(): Promise<boolean> {
  if (process.env.AV_SCANNING_ENABLED === 'true') return true
  const res = await fetch(`${apiBase}/api/v1/admin/av-scan/status`)
  return res.status === 401 || res.status === 403
}

export async function isStatePrivacyEnabled(): Promise<boolean> {
  if (process.env.FEATURE_STATE_PRIVACY === 'true' || process.env.STATE_PRIVACY_ENABLED === 'true') return true
  const res = await fetch(`${apiBase}/api/v1/compliance/state/disclosure/00000000-0000-0000-0000-000000000001`)
  return res.status === 401
}

export async function isCCPAEnabled(): Promise<boolean> {
  if (process.env.FEATURE_CCPA_MODULE === 'true' || process.env.CCPA_MODULE_ENABLED === 'true') return true
  const res = await fetch(`${apiBase}/api/v1/compliance/ccpa/opt-out`)
  return res.status === 401
}

export async function isFERPAEnabled(): Promise<boolean> {
  if (process.env.FEATURE_FERPA_WORKFLOW === 'true' || process.env.FERPA_WORKFLOW_ENABLED === 'true') return true
  const res = await fetch(`${apiBase}/api/v1/compliance/ferpa/directory-opt-out`)
  return res.status === 401
}

export async function isBackupEnabled(): Promise<boolean> {
  if (process.env.BACKUP_MODULE_ENABLED === 'true' || process.env.FEATURE_BACKUP_MODULE === 'true') return true
  const res = await fetch(`${apiBase}/api/v1/internal/ops/backup-status`)
  return res.status === 401
}
