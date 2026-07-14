import { type FormEvent, useCallback, useEffect, useState } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { LearnerProfilePanel } from '../../components/settings/learner-profile-panel'
import { settingsViewFromPathname } from '../../components/layout/side-nav-path-utils'
import { usePlatformScimEnabled } from '../../hooks/use-platform-scim-enabled'
import { ImageModelPicker } from '../../components/image-model-picker'
import { RequirePermission } from '../../components/require-permission'
import { LtiToolsSettingsPanel } from '../../components/settings/lti-tools-settings-panel'
import { OrganizationsPanel } from '../../components/settings/organizations-panel'
import { OrgUnitsPanel } from '../../components/settings/org-units-panel'
import { TermsSettingsPanel } from '../../components/settings/terms-settings-panel'
import { PlatformSettingsPanel } from '../../components/settings/platform-settings-panel'
import { ScimSettingsPanel } from '../../components/settings/scim-settings-panel'
import { CloudProvidersPanel } from '../../components/settings/cloud-providers-panel'
import { LRSSettingsPanel } from '../../components/settings/lrs-settings-panel'
import { OERProvidersPanel } from '../../components/settings/oer-providers-panel'
import { TranscriptsSettingsPanel } from '../../components/settings/transcripts-settings-panel'
import { AdvisingSettingsPanel } from '../../components/settings/advising-settings-panel'
import { oerLibraryEnabled } from '../../lib/oer-api'
import { xapiEmissionFeatureEnabled } from '../../lib/platform-features'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { RolesPermissionsPanel } from '../../components/settings/roles-permissions-panel'
import { usePermissions } from '../../context/use-permissions'
import {
  PERM_RBAC_MANAGE,
  PERM_TENANT_ORG_UNITS_ADMIN,
} from '../../lib/rbac-api'
import { AccountSettingsView } from '../../components/settings/account-settings-view'
import { IntegrationsAccessKeysPanel } from '../../components/settings/integrations-access-keys-panel'
import { CalendarSubscriptionsPanel } from '../../components/settings/calendar-subscriptions-panel'
import { AdminServiceTokensPanel } from '../../components/settings/admin-service-tokens-panel'
import { IntegrationsMcpPanel } from '../../components/settings/integrations-mcp-panel'
import { NotificationPreferencesPanel } from '../../components/settings/notification-preferences-panel'
import { LearningGoalsPanel } from '../../components/onboarding/learning-goals-panel'

import { AiGovernancePanel } from '../../components/settings/ai-governance-panel'
import { AiProviderSettingsPanel } from '../../components/settings/ai-provider-settings-panel'
import { ArchivedCoursesPanel } from '../../components/settings/archived-courses-panel'
import { PeoplePanel } from '../../components/settings/people-panel'
import { CoursesPanel } from '../../components/settings/courses-panel'
import { IntroCoursePanel } from '../../components/settings/intro-course-panel'
import { FeedbackAdminPanel } from '../../components/settings/feedback-admin-panel'
import { AiReportsPanel } from '../../components/settings/ai-reports-panel'
import { LmsPage } from './lms-page'
import OrgBranding from './admin/org-branding'
import { FALLBACK_IMAGE_MODEL_OPTIONS, FALLBACK_TEXT_MODEL_OPTIONS } from '../../lib/ai-models'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { PLATFORM_SECRET_PLACEHOLDER } from '../../lib/platform-settings'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

function isSystemSettingsPath(pathname: string): boolean {
  if (pathname.startsWith('/settings/ai/')) return true
  if (pathname.startsWith('/settings/feedback')) return true
  return (
    pathname === '/settings/roles' ||
    pathname === '/settings/lti-tools' ||
    pathname === '/settings/platform' ||
    pathname === '/settings/organizations' ||
    pathname === '/settings/org-units' ||
    pathname === '/settings/terms' ||
    pathname === '/settings/org-branding' ||
    pathname === '/settings/scim-provisioning' ||
    pathname === '/settings/cloud-providers' ||
    pathname === '/settings/lrs-integrations' ||
    pathname === '/settings/oer-providers' ||
    pathname === '/settings/transcripts' ||
    pathname === '/settings/advising' ||
    pathname === '/settings/archive' ||
    pathname === '/settings/people' ||
    pathname === '/settings/courses' ||
    pathname === '/settings/intro-course'
  )
}

type SystemPromptItem = {
  key: string
  label: string
  content: string
  updatedAt: string
}

type AiModelOption = {
  id: string
  name: string
  contextLength?: number | null
  inputPricePerMillionUsd?: number | null
  outputPricePerMillionUsd?: number | null
  modalitiesSummary?: string | null
}

function fallbackImageModels(): AiModelOption[] {
  return FALLBACK_IMAGE_MODEL_OPTIONS.map((o) => ({
    id: o.id,
    name: o.label,
    contextLength: null,
    inputPricePerMillionUsd: null,
    outputPricePerMillionUsd: null,
    modalitiesSummary: null,
  }))
}

function fallbackTextModels(): AiModelOption[] {
  return FALLBACK_TEXT_MODEL_OPTIONS.map((o) => ({
    id: o.id,
    name: o.label,
    contextLength: null,
    inputPricePerMillionUsd: null,
    outputPricePerMillionUsd: null,
    modalitiesSummary: null,
  }))
}

type ModelKind = 'image' | 'text'

async function fetchModelsForKind(kind: ModelKind): Promise<{
  models: AiModelOption[]
  fromApi: boolean
  configured: boolean
}> {
  const modelsRes = await authorizedFetch(`/api/v1/settings/ai/models?kind=${kind}`)
  const modelsRaw: unknown = await modelsRes.json().catch(() => ({}))
  if (!modelsRes.ok) {
    throw new Error(readApiErrorMessage(modelsRaw))
  }
  const list = modelsRaw as {
    configured?: boolean
    models?: AiModelOption[]
  }
  const apiModels = list.models ?? []
  const configured = list.configured === true
  if (apiModels.length > 0) {
    return { models: apiModels, fromApi: true, configured }
  }
  return {
    models: kind === 'image' ? fallbackImageModels() : fallbackTextModels(),
    fromApi: false,
    configured,
  }
}

export default function Settings() {
  const location = useLocation()
  const { allows, loading: permLoading } = usePermissions()
  const activeView = settingsViewFromPathname(location.pathname)
  const canManageRbac = !permLoading && allows(PERM_RBAC_MANAGE)
  const { scimEnabled: platformScimEnabled, loading: platformScimFlagLoading } = usePlatformScimEnabled(
    canManageRbac && activeView === 'scim-provisioning',
  )
  const {
    ffTranscripts,
    ffAdvisingIntegration,
    learnerProfileEnabled,
    loading: featuresLoading,
  } = usePlatformFeatures()
  const [systemPrompts, setSystemPrompts] = useState<SystemPromptItem[]>([])
  const [systemPromptKey, setSystemPromptKey] = useState('')
  const [systemPromptDraft, setSystemPromptDraft] = useState('')
  const [systemPromptsLoading, setSystemPromptsLoading] = useState(false)
  const [systemPromptsSaving, setSystemPromptsSaving] = useState(false)
  const [systemPromptsError, setSystemPromptsError] = useState<string | null>(null)
  const [systemPromptsMessage, setSystemPromptsMessage] = useState<string | null>(null)

  const [imageModelId, setImageModelId] = useState('')
  const [courseSetupModelId, setCourseSetupModelId] = useState('')
  const [notebookFlashcardsModelId, setNotebookFlashcardsModelId] = useState('')
  const [vibeActivityModelId, setVibeActivityModelId] = useState('')
  const [graderAgentModelId, setGraderAgentModelId] = useState('')
  const [aiLoading, setAiLoading] = useState(true)
  const [aiSaving, setAiSaving] = useState(false)
  const [aiMessage, setAiMessage] = useState<string | null>(null)
  const [aiError, setAiError] = useState<string | null>(null)

  const [imageModels, setImageModels] = useState<AiModelOption[]>([])
  const [textModels, setTextModels] = useState<AiModelOption[]>([])
  const [imageModelsFromApi, setImageModelsFromApi] = useState(false)
  const [textModelsFromApi, setTextModelsFromApi] = useState(false)
  const [modelsConfigured, setModelsConfigured] = useState(false)
  const [modelsError, setModelsError] = useState<string | null>(null)
  const [modelsRefreshing, setModelsRefreshing] = useState(false)
  const [openRouterApiKey, setOpenRouterApiKey] = useState('')
  const [openRouterApiKeyBaseline, setOpenRouterApiKeyBaseline] = useState('')

  const loadModels = useCallback(async () => {
    setModelsError(null)
    try {
      const [img, txt] = await Promise.all([
        fetchModelsForKind('image'),
        fetchModelsForKind('text'),
      ])
      setModelsConfigured(img.configured)
      setImageModels(img.models)
      setImageModelsFromApi(img.fromApi)
      setTextModels(txt.models)
      setTextModelsFromApi(txt.fromApi)
    } catch (e) {
      setModelsError(e instanceof Error ? e.message : 'Could not load models.')
      setImageModels(fallbackImageModels())
      setTextModels(fallbackTextModels())
      setImageModelsFromApi(false)
      setTextModelsFromApi(false)
      setModelsConfigured(false)
    }
  }, [])

  const refreshModels = useCallback(async () => {
    setModelsRefreshing(true)
    await loadModels()
    setModelsRefreshing(false)
  }, [loadModels])

  const loadSystemPrompts = useCallback(async () => {
    setSystemPromptsLoading(true)
    setSystemPromptsError(null)
    try {
      const res = await authorizedFetch('/api/v1/settings/system-prompts')
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setSystemPromptsError(readApiErrorMessage(raw))
        return
      }
      const data = raw as { prompts?: SystemPromptItem[] }
      const list = data.prompts ?? []
      setSystemPrompts(list)
      if (list.length > 0) {
        setSystemPromptKey((prev) => {
          const nextKey = list.some((p) => p.key === prev) ? prev : list[0].key
          const row = list.find((p) => p.key === nextKey)
          if (row) setSystemPromptDraft(row.content)
          return nextKey
        })
      }
    } catch {
      setSystemPromptsError('Could not load system prompts.')
    } finally {
      setSystemPromptsLoading(false)
    }
  }, [])

  useEffect(() => {
    if (activeView !== 'ai-prompts') return
    if (permLoading || !allows(PERM_RBAC_MANAGE)) return
    void loadSystemPrompts()
  }, [activeView, allows, loadSystemPrompts, permLoading])

  useEffect(() => {
    if (activeView !== 'ai-models' || !canManageRbac) return
    let cancelled = false
    ;(async () => {
      setAiLoading(true)
      setAiError(null)
      setModelsError(null)
      try {
        const settingsRes = await authorizedFetch('/api/v1/settings/ai')
        const settingsRaw: unknown = await settingsRes.json().catch(() => ({}))
        if (!settingsRes.ok) {
          if (!cancelled) setAiError(readApiErrorMessage(settingsRaw))
        } else {
          const data = settingsRaw as {
            imageModelId?: string
            courseSetupModelId?: string
            notebookFlashcardsModelId?: string
            vibeActivityModelId?: string
            graderAgentModelId?: string
            openRouterApiKey?: string
          }
          if (!cancelled && data.imageModelId) setImageModelId(data.imageModelId)
          if (!cancelled && data.courseSetupModelId) setCourseSetupModelId(data.courseSetupModelId)
          if (!cancelled && data.notebookFlashcardsModelId) setNotebookFlashcardsModelId(data.notebookFlashcardsModelId)
          if (!cancelled && data.vibeActivityModelId) setVibeActivityModelId(data.vibeActivityModelId)
          if (!cancelled && data.graderAgentModelId) setGraderAgentModelId(data.graderAgentModelId)
          if (!cancelled) {
            const key = data.openRouterApiKey ?? ''
            setOpenRouterApiKey(key)
            setOpenRouterApiKeyBaseline(key)
          }
        }
        if (!cancelled) await loadModels()
      } catch {
        if (!cancelled) {
          setAiError('Could not load AI settings.')
          setImageModels(fallbackImageModels())
          setTextModels(fallbackTextModels())
          setImageModelsFromApi(false)
          setTextModelsFromApi(false)
          setModelsConfigured(false)
        }
      } finally {
        if (!cancelled) setAiLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [activeView, canManageRbac, loadModels])

  async function onSaveAi(e: FormEvent) {
    e.preventDefault()
    setAiSaving(true)
    setAiMessage(null)
    setAiError(null)
    try {
      const body: Record<string, unknown> = {
        imageModelId,
        courseSetupModelId,
        notebookFlashcardsModelId,
        vibeActivityModelId,
        graderAgentModelId,
      }
      const keyTrimmed = openRouterApiKey.trim()
      const keyBaselineTrimmed = openRouterApiKeyBaseline.trim()
      if (keyTrimmed !== keyBaselineTrimmed) {
        const v = keyTrimmed
        if (v && v !== PLATFORM_SECRET_PLACEHOLDER) {
          body.openRouterApiKey = v
        }
        if (
          keyBaselineTrimmed === PLATFORM_SECRET_PLACEHOLDER &&
          v === '' &&
          openRouterApiKey !== openRouterApiKeyBaseline
        ) {
          body.clearOpenRouterApiKey = true
        }
      }

      const res = await authorizedFetch('/api/v1/settings/ai', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setAiError(readApiErrorMessage(raw))
        return
      }
      const data = raw as {
        imageModelId?: string
        courseSetupModelId?: string
        notebookFlashcardsModelId?: string
        vibeActivityModelId?: string
        graderAgentModelId?: string
        openRouterApiKey?: string
      }
      if (data.imageModelId) setImageModelId(data.imageModelId)
      if (data.courseSetupModelId) setCourseSetupModelId(data.courseSetupModelId)
      if (data.notebookFlashcardsModelId) setNotebookFlashcardsModelId(data.notebookFlashcardsModelId)
      if (data.vibeActivityModelId) setVibeActivityModelId(data.vibeActivityModelId)
      if (data.graderAgentModelId) setGraderAgentModelId(data.graderAgentModelId)
      if (data.openRouterApiKey !== undefined) {
        setOpenRouterApiKey(data.openRouterApiKey)
        setOpenRouterApiKeyBaseline(data.openRouterApiKey)
      }
      setAiMessage('Saved.')
      toastSaveOk('AI settings saved')
      await loadModels()
    } catch {
      setAiError('Could not save settings.')
      toastMutationError('Could not save AI settings.')
    } finally {
      setAiSaving(false)
    }
  }

  const saveDisabled =
    aiSaving || !imageModelId || !courseSetupModelId || !notebookFlashcardsModelId || !vibeActivityModelId

  async function onSaveSystemPrompt(e: FormEvent) {
    e.preventDefault()
    if (!systemPromptKey.trim()) return
    setSystemPromptsSaving(true)
    setSystemPromptsError(null)
    setSystemPromptsMessage(null)
    try {
      const res = await authorizedFetch(
        `/api/v1/settings/system-prompts/${encodeURIComponent(systemPromptKey)}`,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ content: systemPromptDraft }),
        },
      )
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setSystemPromptsError(readApiErrorMessage(raw))
        return
      }
      const row = raw as SystemPromptItem
      setSystemPrompts((prev) =>
        prev.map((p) =>
          p.key === row.key
            ? { ...p, content: row.content, updatedAt: row.updatedAt }
            : p,
        ),
      )
      setSystemPromptsMessage('Saved.')
      toastSaveOk('System prompt saved')
    } catch {
      setSystemPromptsError('Could not save system prompt.')
      toastMutationError('Could not save system prompt.')
    } finally {
      setSystemPromptsSaving(false)
    }
  }

  function onSystemPromptKeyChange(key: string) {
    setSystemPromptKey(key)
    const row = systemPrompts.find((p) => p.key === key)
    if (row) setSystemPromptDraft(row.content)
  }

  if (location.pathname === '/settings/org-roles') {
    return <Navigate to="/settings/roles" replace />
  }

  if (permLoading && isSystemSettingsPath(location.pathname)) {
    return (
      <LmsPage title="Settings" description="Account and learning preferences.">
        <p className="mt-8 text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
      </LmsPage>
    )
  }
  if (!permLoading && isSystemSettingsPath(location.pathname)) {
    const onOrgUnits = location.pathname === '/settings/org-units'
    const onTerms = location.pathname === '/settings/terms'
    const onOrgBranding = location.pathname === '/settings/org-branding'
    const hasRbac = allows(PERM_RBAC_MANAGE)
    const hasUnitAdmin = allows(PERM_TENANT_ORG_UNITS_ADMIN)
    if (!hasRbac && !((onOrgUnits || onTerms || onOrgBranding) && hasUnitAdmin)) {
      return <Navigate to="/settings/account" replace />
    }
  }

  if (activeView === 'scim-provisioning' && canManageRbac && platformScimFlagLoading) {
    return (
      <LmsPage title="Settings" description="Account and learning preferences.">
        <p className="mt-8 text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
      </LmsPage>
    )
  }
  if (activeView === 'scim-provisioning' && canManageRbac && !platformScimEnabled) {
    return <Navigate to="/settings/platform" replace />
  }
  if (activeView === 'transcripts' && canManageRbac && !featuresLoading && !ffTranscripts) {
    return <Navigate to="/settings/platform" replace />
  }
  if (activeView === 'advising' && canManageRbac && !featuresLoading && !ffAdvisingIntegration) {
    return <Navigate to="/settings/platform" replace />
  }
  if (activeView === 'learner-profile' && !featuresLoading && !learnerProfileEnabled) {
    return <Navigate to="/settings/account" replace />
  }

  return (
    <LmsPage title="Settings" description="Account and learning preferences.">
      <div
        className={`mt-8 ${
          activeView === 'roles' ||
          activeView === 'lti-tools' ||
          activeView === 'platform' ||
          activeView === 'organizations' ||
          activeView === 'org-units' ||
          activeView === 'terms' ||
          activeView === 'org-branding' ||
          activeView === 'scim-provisioning' ||
          activeView === 'cloud-providers' ||
          activeView === 'lrs-integrations' ||
          activeView === 'oer-providers' ||
          activeView === 'transcripts' ||
          activeView === 'advising' ||
          activeView === 'archive' ||
          activeView === 'people' ||
          activeView === 'courses' ||
          activeView === 'intro-course' ||
          activeView === 'feedback'
            ? 'max-w-4xl'
            : activeView === 'integrations'
              ? 'max-w-3xl'
            : activeView === 'ai-prompts'
              ? 'max-w-3xl'
              : activeView === 'ai-reports'
                ? 'max-w-4xl'
              : activeView === 'account'
                ? 'max-w-3xl'
              : activeView === 'learner-profile'
                ? 'max-w-3xl'
              : 'max-w-xl'
        }`}
      >
        {activeView === 'ai-models' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Models</h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Choose models for course setup (text) and for generating course hero images. Lists are
              loaded from{' '}
              <a
                href="https://openrouter.ai/docs/api/api-reference/models/get-models"
                className="font-medium text-indigo-600 hover:text-indigo-500"
                target="_blank"
                rel="noreferrer"
              >
                OpenRouter&apos;s models API
              </a>{' '}
              (text-capable and image-capable models). Generation requires an OpenRouter API key
              configured below.
            </p>

            {aiLoading && <p className="mt-4 text-sm text-slate-500">Loading…</p>}
            {aiError && (
              <p className="mt-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800">
                {aiError}
              </p>
            )}

            {!modelsConfigured && !aiLoading && (
              <p className="mt-4 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-100">
                Add your OpenRouter API key below and save, so AI features can call OpenRouter.
              </p>
            )}

            {modelsError && !aiLoading && (
              <p className="mt-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800">
                {modelsError} Showing a static fallback list.
              </p>
            )}

            {!imageModelsFromApi && modelsConfigured && !aiLoading && !modelsError && (
              <p className="mt-4 text-sm text-slate-500">No image models returned from OpenRouter; using fallback IDs.</p>
            )}

            {!textModelsFromApi && modelsConfigured && !aiLoading && !modelsError && (
              <p className="mt-4 text-sm text-slate-500">No text models returned from OpenRouter; using fallback IDs.</p>
            )}

            {!aiLoading && (
              <form className="mt-6 space-y-5" onSubmit={onSaveAi}>
                <div>
                  <label
                    htmlFor="openrouter-api-key"
                    className="block text-sm font-medium text-slate-700 dark:text-neutral-200"
                  >
                    OpenRouter API key
                  </label>
                  <input
                    id="openrouter-api-key"
                    type="password"
                    autoComplete="off"
                    value={openRouterApiKey}
                    onChange={(e) => setOpenRouterApiKey(e.target.value)}
                    placeholder={PLATFORM_SECRET_PLACEHOLDER}
                    disabled={aiSaving}
                    className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 font-mono text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                  />
                  <p className="mt-1.5 text-xs text-slate-500 dark:text-neutral-400">
                    Platform-wide key for AI generation. Leave unchanged to keep the current key; clear
                    the field and save to remove it.
                  </p>
                </div>

                <div>
                  <ImageModelPicker
                    id="course-setup-model"
                    label="Course setup model"
                    models={textModels}
                    value={courseSetupModelId}
                    onChange={setCourseSetupModelId}
                    disabled={aiSaving}
                    onRefresh={refreshModels}
                    refreshing={modelsRefreshing}
                  />
                  <p className="mt-1.5 text-xs text-slate-500">
                    Text-to-text model used when setting up course structure and content. Each option
                    shows the display name, model id, then modalities, context window, and
                    input/output price per 1M tokens (USD). Use{' '}
                    <span className="font-medium">Refresh list</span> to reload from OpenRouter.
                  </p>
                </div>

                <div>
                  <ImageModelPicker
                    id="notebook-flashcards-model"
                    label="Notebook flashcards model"
                    models={textModels}
                    value={notebookFlashcardsModelId}
                    onChange={setNotebookFlashcardsModelId}
                    disabled={aiSaving}
                    onRefresh={refreshModels}
                    refreshing={modelsRefreshing}
                  />
                  <p className="mt-1.5 text-xs text-slate-500">
                    Text-to-text model used when generating AI study flashcards from notebook notes.
                  </p>
                </div>

                <div>
                  <ImageModelPicker
                    id="vibe-activity-model"
                    label="Vibe activity model"
                    models={textModels}
                    value={vibeActivityModelId}
                    onChange={setVibeActivityModelId}
                    disabled={aiSaving}
                    onRefresh={refreshModels}
                    refreshing={modelsRefreshing}
                  />
                  <p className="mt-1.5 text-xs text-slate-500">
                    Text-to-text model used when generating interactive HTML vibe activities for courses.
                  </p>
                </div>

                <div>
                  <ImageModelPicker
                    id="grader-agent-model"
                    label="Grading agent model"
                    models={textModels}
                    value={graderAgentModelId}
                    onChange={setGraderAgentModelId}
                    disabled={aiSaving}
                    onRefresh={refreshModels}
                    refreshing={modelsRefreshing}
                  />
                  <p className="mt-1.5 text-xs text-slate-500">
                    Text-to-text model used when instructors dry-run or batch-run the SpeedGrader grading agent.
                  </p>
                </div>

                <div>
                  <ImageModelPicker
                    id="image-model"
                    label="Image model"
                    models={imageModels}
                    value={imageModelId}
                    onChange={setImageModelId}
                    disabled={aiSaving}
                    onRefresh={refreshModels}
                    refreshing={modelsRefreshing}
                  />
                  <p className="mt-1.5 text-xs text-slate-500">
                    Used when you generate course images. Each option shows the display name, model id,
                    then modalities, context window, and input/output price per 1M tokens (USD). Use{' '}
                    <span className="font-medium">Refresh list</span> to reload from OpenRouter.
                  </p>
                </div>

                {aiMessage && (
                  <p className="text-sm text-emerald-700" role="status">
                    {aiMessage}
                  </p>
                )}

                <button
                  type="submit"
                  disabled={saveDisabled}
                  className="rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white dark:shadow-none"
                >
                  {aiSaving ? 'Saving…' : 'Save'}
                </button>
              </form>
            )}
          </div>
        )}

        {activeView === 'ai-prompts' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">System Prompts</h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Edit platform system prompts used by AI features. Changes are audited.
            </p>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800/50 dark:text-neutral-300">
                  You need permission to manage system prompts (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              {systemPromptsLoading && (
                <p className="mt-4 text-sm text-slate-500">Loading…</p>
              )}
              {systemPromptsError && (
                <p className="mt-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
                  {systemPromptsError}
                </p>
              )}
              {!systemPromptsLoading && systemPrompts.length > 0 && (
                <form className="mt-6 space-y-4" onSubmit={onSaveSystemPrompt}>
                  <div>
                    <label htmlFor="system-prompt-select" className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-neutral-200">
                      Prompt
                    </label>
                    <select
                      id="system-prompt-select"
                      value={systemPromptKey}
                      onChange={(e) => onSystemPromptKeyChange(e.target.value)}
                      className="w-full rounded-xl border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                    >
                      {systemPrompts.map((p) => (
                        <option key={p.key} value={p.key}>
                          {p.label}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label htmlFor="system-prompt-body" className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-neutral-200">
                      Content
                    </label>
                    <textarea
                      id="system-prompt-body"
                      value={systemPromptDraft}
                      onChange={(e) => setSystemPromptDraft(e.target.value)}
                      rows={12}
                      spellCheck={false}
                      className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 font-mono text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                    />
                  </div>
                  {systemPromptsMessage && (
                    <p className="text-sm text-emerald-700 dark:text-emerald-400" role="status">
                      {systemPromptsMessage}
                    </p>
                  )}
                  <button
                    type="submit"
                    disabled={systemPromptsSaving || !systemPromptKey}
                    className="rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white dark:shadow-none"
                  >
                    {systemPromptsSaving ? 'Saving…' : 'Save'}
                  </button>
                </form>
              )}
              {!systemPromptsLoading && systemPrompts.length === 0 && !systemPromptsError && (
                <p className="mt-4 text-sm text-slate-500">No system prompts are registered.</p>
              )}
            </RequirePermission>
          </div>
        )}

        {activeView === 'ai-reports' && (
          <RequirePermission
            permission={PERM_RBAC_MANAGE}
            fallback={
              <p className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800/50 dark:text-neutral-300">
                You need permission to view AI reports (
                <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
              </p>
            }
          >
            <AiReportsPanel />
          </RequirePermission>
        )}

        {activeView === 'account' && <AccountSettingsView />}

        {activeView === 'notifications' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Notifications</h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Control which events send you email and whether they arrive instantly or in a daily digest.
            </p>
            <NotificationPreferencesPanel />
            <LearningGoalsPanel />
          </div>
        )}

        {activeView === 'integrations' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Integrations</h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Create access keys for API tools and configure MCP so AI agents can work with your Lextures data.
            </p>
            <IntegrationsAccessKeysPanel />
            <CalendarSubscriptionsPanel />
            <AdminServiceTokensPanel />
            <IntegrationsMcpPanel />
          </div>
        )}

        {activeView === 'learner-profile' && learnerProfileEnabled && <LearnerProfilePanel />}

        {activeView === 'roles' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900">Roles and Permissions</h2>
            <p className="mt-1 text-sm text-slate-500">
              Define permission strings and assign them to roles. Route and UI checks use the same
              matching rules as the server.
            </p>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600">
                  You do not have permission to manage roles and permissions (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <RolesPermissionsPanel />
            </RequirePermission>
          </div>
        )}

        {activeView === 'lti-tools' && (
          <RequirePermission
            permission={PERM_RBAC_MANAGE}
            fallback={
              <div>
                <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">LTI tools</h2>
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
                  You do not have permission to manage LTI registrations (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              </div>
            }
          >
            <LtiToolsSettingsPanel />
          </RequirePermission>
        )}

        {activeView === 'scim-provisioning' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">SCIM provisioning</h2>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
                  You need permission to manage SCIM provisioning (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <ScimSettingsPanel />
            </RequirePermission>
          </div>
        )}

        {activeView === 'cloud-providers' && (
          <div>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
                  You need permission to manage cloud provider settings (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <CloudProvidersPanel />
            </RequirePermission>
          </div>
        )}

        {activeView === 'lrs-integrations' && xapiEmissionFeatureEnabled() && (
          <div>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
                  You need permission to manage LRS integrations (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <LRSSettingsPanel />
            </RequirePermission>
          </div>
        )}

        {activeView === 'oer-providers' && oerLibraryEnabled() && (
          <div>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
                  You need permission to manage OER provider settings (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <OERProvidersPanel />
            </RequirePermission>
          </div>
        )}

        {activeView === 'transcripts' && ffTranscripts && (
          <div>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
                  You need permission to manage transcript settings (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <TranscriptsSettingsPanel />
            </RequirePermission>
          </div>
        )}

        {activeView === 'advising' && ffAdvisingIntegration && (
          <div>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
                  You need permission to manage advising settings (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <AdvisingSettingsPanel />
            </RequirePermission>
          </div>
        )}

        {activeView === 'platform' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Global platform</h2>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
                  You need permission to edit platform configuration (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <PlatformSettingsPanel />
            </RequirePermission>
          </div>
        )}

        {activeView === 'org-units' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Org structure</h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Schools, colleges, and departments within your organization.
            </p>
            <OrgUnitsPanel />
          </div>
        )}

        {activeView === 'terms' && (
          <div className="mt-2">
            <TermsSettingsPanel />
          </div>
        )}

        {activeView === 'org-branding' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
              Organization branding
            </h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Logo, colors, optional custom domain, and email sender display name.
            </p>
            <OrgBranding />
            <AiGovernancePanel />
            <AiProviderSettingsPanel />
          </div>
        )}

        {activeView === 'archive' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Archive</h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Review archived courses, restore them to the catalog, or permanently delete them.
            </p>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800/50 dark:text-neutral-300">
                  You need permission to manage archived courses (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <ArchivedCoursesPanel />
            </RequirePermission>
          </div>
        )}

        {activeView === 'people' && (
          <div>
            <div className="flex flex-wrap items-end justify-between gap-3">
              <div>
                <h2 className="text-xl font-semibold tracking-tight text-slate-900 dark:text-neutral-100">
                  People
                </h2>
                <p className="mt-1 max-w-2xl text-sm leading-relaxed text-slate-500 dark:text-neutral-400">
                  Search, invite, suspend, and manage user accounts across the platform.
                </p>
              </div>
            </div>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800/50 dark:text-neutral-300">
                  You need permission to manage people (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <PeoplePanel />
            </RequirePermission>
          </div>
        )}

        {activeView === 'courses' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Courses</h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Search courses across the platform and open them with instructor access to manage content and enrollments.
            </p>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800/50 dark:text-neutral-300">
                  You need permission to manage courses (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <CoursesPanel />
            </RequirePermission>
          </div>
        )}

        {activeView === 'intro-course' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Intro course</h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Govern the platform-wide Welcome to Lextures onboarding course: enable/disable, re-sync content, backfill
              enrollments, and read completion analytics.
            </p>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800/50 dark:text-neutral-300">
                  You need permission to manage the intro course (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <IntroCoursePanel />
            </RequirePermission>
          </div>
        )}

        {activeView === 'feedback' && (
          <RequirePermission
            permission={PERM_RBAC_MANAGE}
            fallback={
              <div>
                <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Feedback</h2>
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800/50 dark:text-neutral-300">
                  You need permission to review feedback (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              </div>
            }
          >
            <FeedbackAdminPanel />
          </RequirePermission>
        )}

        {activeView === 'organizations' && (
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Organizations</h2>
            <RequirePermission
              permission={PERM_RBAC_MANAGE}
              fallback={
                <p className="mt-6 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
                  You need permission to manage organizations (
                  <code className="font-mono text-xs">{PERM_RBAC_MANAGE}</code>).
                </p>
              }
            >
              <OrganizationsPanel />
            </RequirePermission>
          </div>
        )}
      </div>

    </LmsPage>
  )
}
