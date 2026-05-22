import { useCallback, useEffect, useId, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ExternalLink } from 'lucide-react'
import { usePermissions } from '../../context/use-permissions'
import {
  fetchModuleExternalLink,
  patchModuleExternalLink,
  type ModuleExternalLinkPayload,
} from '../../lib/courses-api'
import { recordLastVisitedModuleItem } from '../../lib/last-visited-module-item'
import { permCourseItemCreate } from '../../lib/rbac-api'
import {
  GoogleDrivePicker,
  OneDrivePicker,
  DropboxPicker,
  type PickedFile,
} from '../../services/cloud-picker'
import { LmsPage } from './lms-page'

const PROVIDER_LABELS: Record<string, string> = {
  google_drive: 'Google Drive',
  onedrive: 'OneDrive',
  dropbox: 'Dropbox',
  oer_commons: 'OER Commons',
  merlot: 'MERLOT',
  openstax: 'OpenStax',
  url: 'External URL',
}

const PROVIDER_COLORS: Record<string, string> = {
  google_drive: 'bg-blue-50 text-blue-700 border-blue-200/80 dark:bg-blue-950/55 dark:text-blue-300 dark:border-blue-500/40',
  onedrive: 'bg-sky-50 text-sky-700 border-sky-200/80 dark:bg-sky-950/55 dark:text-sky-300 dark:border-sky-500/40',
  dropbox: 'bg-indigo-50 text-indigo-700 border-indigo-200/80 dark:bg-indigo-950/55 dark:text-indigo-300 dark:border-indigo-500/40',
}

function ProviderBadge({ provider }: { provider: string }) {
  const label = PROVIDER_LABELS[provider] ?? provider
  const color = PROVIDER_COLORS[provider] ?? 'bg-slate-50 text-slate-700 border-slate-200/80 dark:bg-neutral-800 dark:text-neutral-300 dark:border-neutral-600'
  return (
    <span className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium ${color}`}>
      {label}
    </span>
  )
}

type CloudButton = {
  label: string
  provider: 'google_drive' | 'onedrive' | 'dropbox'
  pick: () => Promise<PickedFile | null>
}

function CloudPickerButtons({
  onPicked,
  disabled,
}: {
  onPicked: (file: PickedFile) => void
  disabled?: boolean
}) {
  const [picking, setPicking] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const buttons: CloudButton[] = [
    {
      label: 'Google Drive',
      provider: 'google_drive',
      pick: () => new GoogleDrivePicker('', '').pick(),
    },
    {
      label: 'OneDrive',
      provider: 'onedrive',
      pick: () => new OneDrivePicker('').pick(),
    },
    {
      label: 'Dropbox',
      provider: 'dropbox',
      pick: () => new DropboxPicker().pick(),
    },
  ]

  async function handlePick(btn: CloudButton) {
    setError(null)
    setPicking(btn.provider)
    try {
      const file = await btn.pick()
      if (file) onPicked(file)
    } catch (e) {
      setError(e instanceof Error ? e.message : `Could not open ${btn.label} picker.`)
    } finally {
      setPicking(null)
    }
  }

  return (
    <div>
      <p className="mb-2 text-xs font-medium text-slate-600 dark:text-neutral-300">
        Or link from cloud storage
      </p>
      <div className="flex flex-wrap gap-2">
        {buttons.map((btn) => (
          <button
            key={btn.provider}
            type="button"
            disabled={disabled || picking !== null}
            onClick={() => void handlePick(btn)}
            aria-haspopup="dialog"
            className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700"
          >
            {picking === btn.provider ? 'Opening…' : btn.label}
          </button>
        ))}
      </div>
      {error && (
        <p className="mt-2 text-xs text-rose-700 dark:text-rose-300" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}

export default function CourseModuleExternalLinkPage() {
  const { courseCode, itemId } = useParams<{ courseCode: string; itemId: string }>()
  const { allows, loading: permLoading } = usePermissions()
  const urlFieldId = useId()

  const [data, setData] = useState<ModuleExternalLinkPayload | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [draftUrl, setDraftUrl] = useState('')
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [autoOpenDone, setAutoOpenDone] = useState(false)

  const canEdit = Boolean(
    courseCode && itemId && !permLoading && allows(permCourseItemCreate(courseCode)),
  )

  const load = useCallback(async () => {
    if (!courseCode || !itemId) return
    setLoading(true)
    setLoadError(null)
    try {
      const row = await fetchModuleExternalLink(courseCode, itemId)
      setData(row)
      setDraftUrl(row.url)
      recordLastVisitedModuleItem(courseCode, {
        itemId,
        kind: 'external_link',
        title: row.title,
      })
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : 'Could not load this link.')
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [courseCode, itemId])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    setAutoOpenDone(false)
  }, [itemId, courseCode])

  useEffect(() => {
    if (loading || !data?.url || canEdit || autoOpenDone) return
    const u = data.url.trim()
    if (!u) return
    setAutoOpenDone(true)
    try {
      window.open(u, '_blank', 'noopener,noreferrer')
    } catch {
      /* ignore */
    }
  }, [loading, data, canEdit, autoOpenDone])

  async function onSave(e: React.FormEvent) {
    e.preventDefault()
    if (!courseCode || !itemId) return
    setSaveError(null)
    setSaving(true)
    try {
      const row = await patchModuleExternalLink(courseCode, itemId, { url: draftUrl.trim() })
      setData(row)
      setDraftUrl(row.url)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Could not save URL.')
    } finally {
      setSaving(false)
    }
  }

  async function onCloudPicked(file: PickedFile) {
    if (!courseCode || !itemId) return
    setSaveError(null)
    setSaving(true)
    try {
      const row = await patchModuleExternalLink(courseCode, itemId, {
        url: file.viewUrl,
        provider: file.provider,
        externalId: file.externalId,
        iconUrl: file.iconUrl,
      })
      setData(row)
      setDraftUrl(row.url)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Could not link cloud file.')
    } finally {
      setSaving(false)
    }
  }

  const modulesHref =
    courseCode != null && courseCode !== ''
      ? `/courses/${encodeURIComponent(courseCode)}/modules`
      : '/courses'

  const isCloudLink = data?.provider && data.provider !== 'url'
  const isOerLink = Boolean(data?.oerProvider)
  const openLabel = isOerLink && data?.provider === 'openstax'
    ? 'Open in OpenStax'
    : isCloudLink
      ? `Open in ${PROVIDER_LABELS[data?.provider ?? ''] ?? data?.provider}`
      : 'Open link'

  return (
    <LmsPage title={data?.title ?? 'External link'}>
      <div className="mx-auto max-w-2xl">
        <p className="mb-4 text-sm text-slate-600 dark:text-neutral-400">
          <Link to={modulesHref} className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400">
            ← Modules
          </Link>
        </p>

        {loading && <p className="text-sm text-slate-600 dark:text-neutral-400">Loading…</p>}
        {loadError && (
          <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">
            {loadError}
          </p>
        )}

        {!loading && !loadError && data && (
          <div className="rounded-2xl border border-slate-200/80 bg-white/90 p-5 shadow-sm dark:border-neutral-700 dark:bg-neutral-900/85">
            <div className="flex items-start gap-3">
              <span className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-violet-200/90 bg-violet-50 text-violet-700 dark:border-violet-500/40 dark:bg-violet-950/55 dark:text-violet-200">
                {isCloudLink && data.iconUrl ? (
                  <img src={data.iconUrl} alt="" className="h-5 w-5 object-contain" aria-hidden />
                ) : (
                  <ExternalLink className="h-5 w-5" strokeWidth={2} aria-hidden />
                )}
              </span>
              <div className="min-w-0 flex-1">
                {isCloudLink || isOerLink ? (
                  <div className="flex flex-wrap items-center gap-2">
                    <ProviderBadge provider={data.oerProvider || data.provider} />
                    {data.licenseSpdx ? (
                      <span
                        className="inline-flex items-center rounded-full border border-emerald-200/90 bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-800 dark:border-emerald-500/35 dark:bg-emerald-950 dark:text-emerald-200"
                        aria-label={`Creative Commons ${data.licenseSpdx}`}
                      >
                        {data.licenseSpdx}
                      </span>
                    ) : null}
                  </div>
                ) : null}
                {data.attributionText ? (
                  <p className="mt-2 text-xs text-slate-600 dark:text-neutral-400">
                    <span className="font-medium">Required attribution:</span> {data.attributionText}
                  </p>
                ) : null}
                {!canEdit && data.url ? (
                  <p className="mt-3 text-sm text-slate-600 dark:text-neutral-400">
                    Opening in a new tab… If nothing opened, use the button below (your browser may
                    have blocked the pop-up).
                  </p>
                ) : null}
                <div className="mt-4 flex flex-wrap gap-2">
                  {data.url ? (
                    <a
                      href={data.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center justify-center rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-indigo-500"
                    >
                      {openLabel}
                    </a>
                  ) : (
                    <p className="text-sm text-slate-600 dark:text-neutral-400">
                      No URL has been set for this item yet.
                    </p>
                  )}
                </div>

                {canEdit && (
                  <form className="mt-6 space-y-5 border-t border-slate-200 pt-6 dark:border-neutral-700" onSubmit={onSave}>
                    <div>
                      <label htmlFor={urlFieldId} className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                        Destination URL
                      </label>
                      <input
                        id={urlFieldId}
                        type="url"
                        value={draftUrl}
                        onChange={(e) => setDraftUrl(e.target.value)}
                        disabled={saving}
                        className="mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
                      />
                      {saveError ? (
                        <p className="mt-2 text-sm text-rose-700 dark:text-rose-300" role="status">
                          {saveError}
                        </p>
                      ) : null}
                      <div className="mt-3 flex justify-end">
                        <button
                          type="submit"
                          disabled={saving || !draftUrl.trim()}
                          className="rounded-xl bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-white"
                        >
                          {saving ? 'Saving…' : 'Save URL'}
                        </button>
                      </div>
                    </div>

                    <div className="border-t border-slate-100 pt-4 dark:border-neutral-800">
                      <CloudPickerButtons onPicked={(f) => void onCloudPicked(f)} disabled={saving} />
                    </div>
                  </form>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </LmsPage>
  )
}
