import { useEffect, useId, useState, type ReactNode } from 'react'
import { X } from 'lucide-react'
import { MODULE_REQUIREMENTS_FORM_ID } from '../../components/modules/module-requirements-panel'
import {
  RelativeScheduleBanner,
  ScheduleDatetimeField,
} from '../../components/lms/schedule-datetime-field'

const MODULE_SETTINGS_FORM_ID = 'module-settings-form'

function isoToDatetimeLocalValue(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function datetimeLocalValueToIso(value: string): string | null {
  const t = value.trim()
  if (!t) return null
  const d = new Date(t)
  if (Number.isNaN(d.getTime())) return null
  return d.toISOString()
}

type SettingsTab = 'general' | 'requirements'

type ModuleSettingsModalProps = {
  open: boolean
  initialTitle: string
  initialPublished: boolean
  initialVisibleFrom: string | null
  onClose: () => void
  onSave: (payload: { title: string; published: boolean; visibleFrom: string | null }) => void | Promise<void>
  onDelete?: () => void
  saving?: boolean
  errorMessage?: string | null
  /** When set, shows a Requirements tab with this content. */
  requirementsSection?: ReactNode
  requirementsSaving?: boolean
  /** True while existing requirements are loading for the Requirements tab. */
  requirementsLoading?: boolean
  /** Optional explicit submit for the requirements tab (preferred over form= association). */
  onSaveRequirements?: () => void
  scheduleMode?: string | null
  relativeScheduleAnchorAt?: string | null
}

export function ModuleSettingsModal(props: ModuleSettingsModalProps) {
  if (!props.open) return null
  return <ModuleSettingsModalInner {...props} />
}

function ModuleSettingsModalInner({
  initialTitle,
  initialPublished,
  initialVisibleFrom,
  onClose,
  onSave,
  onDelete,
  saving = false,
  errorMessage,
  requirementsSection,
  requirementsSaving = false,
  requirementsLoading = false,
  onSaveRequirements,
  scheduleMode,
  relativeScheduleAnchorAt,
}: ModuleSettingsModalProps) {
  const titleId = useId()
  const nameInputId = useId()
  const dateInputId = useId()
  const generalTabId = useId()
  const requirementsTabId = useId()
  const generalPanelId = useId()
  const requirementsPanelId = useId()
  const [title, setTitle] = useState(initialTitle)
  const [published, setPublished] = useState(initialPublished)
  const [visibleLocal, setVisibleLocal] = useState(() => isoToDatetimeLocalValue(initialVisibleFrom))
  const [tab, setTab] = useState<SettingsTab>('general')
  const showRequirementsTab = requirementsSection != null
  const busy = saving || requirementsSaving

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key !== 'Escape') return
      if (busy) return
      e.preventDefault()
      onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [busy, onClose])

  const saveDisabled =
    busy ||
    (tab === 'general' && !title.trim()) ||
    (tab === 'requirements' && (!showRequirementsTab || requirementsLoading))
  const saveLabel =
    tab === 'requirements'
      ? requirementsSaving
        ? 'Saving…'
        : 'Save requirements'
      : saving
        ? 'Saving…'
        : 'Save'

  function submitActiveTab() {
    if (busy || saveDisabled) return
    if (tab === 'requirements') {
      if (onSaveRequirements) {
        onSaveRequirements()
        return
      }
      const form = document.getElementById(MODULE_REQUIREMENTS_FORM_ID)
      if (form instanceof HTMLFormElement) {
        form.requestSubmit()
      }
      return
    }
    const form = document.getElementById(MODULE_SETTINGS_FORM_ID)
    if (form instanceof HTMLFormElement) {
      form.requestSubmit()
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      onClick={(e) => {
        if (e.target === e.currentTarget && !busy) onClose()
      }}
    >
      <div className="flex w-full max-w-lg flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl">
        <div className="flex shrink-0 items-center justify-between border-b border-slate-200 px-4 py-3">
          <h3 id={titleId} className="text-sm font-semibold text-slate-900">
            Module settings
          </h3>
          <button
            type="button"
            onClick={() => onClose()}
            disabled={busy}
            className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800 disabled:cursor-not-allowed disabled:opacity-50"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {showRequirementsTab ? (
          <div
            className="flex shrink-0 gap-1 border-b border-slate-200 px-2"
            role="tablist"
            aria-label="Module settings sections"
          >
            <button
              type="button"
              role="tab"
              id={generalTabId}
              aria-controls={generalPanelId}
              aria-selected={tab === 'general'}
              onClick={() => setTab('general')}
              className={`px-3 py-2.5 text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 ${
                tab === 'general'
                  ? 'border-b-2 border-indigo-600 text-indigo-700'
                  : 'text-slate-600 hover:text-slate-900'
              }`}
            >
              General
            </button>
            <button
              type="button"
              role="tab"
              id={requirementsTabId}
              aria-controls={requirementsPanelId}
              aria-selected={tab === 'requirements'}
              onClick={() => setTab('requirements')}
              className={`px-3 py-2.5 text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 ${
                tab === 'requirements'
                  ? 'border-b-2 border-indigo-600 text-indigo-700'
                  : 'text-slate-600 hover:text-slate-900'
              }`}
            >
              Requirements
            </button>
          </div>
        ) : null}

        <div className="p-4">
          <div
            id={generalPanelId}
            role="tabpanel"
            aria-labelledby={generalTabId}
            hidden={showRequirementsTab && tab !== 'general'}
          >
            <form
              id={MODULE_SETTINGS_FORM_ID}
              onSubmit={(e) => {
                e.preventDefault()
                const t = title.trim()
                if (!t || busy) return
                void onSave({
                  title: t,
                  published,
                  visibleFrom: datetimeLocalValueToIso(visibleLocal),
                })
              }}
            >
              <label htmlFor={nameInputId} className="text-xs font-medium text-slate-600">
                Module name
              </label>
              <input
                id={nameInputId}
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="e.g. Week 1 — Introduction"
                autoFocus={!showRequirementsTab || tab === 'general'}
                disabled={busy}
                className="mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 placeholder:text-slate-400 focus:border-indigo-400 focus:ring-2 disabled:cursor-not-allowed disabled:opacity-60"
              />

              <div className="mt-4 flex items-center gap-3">
                <input
                  id="module-settings-published"
                  type="checkbox"
                  checked={published}
                  onChange={(e) => setPublished(e.target.checked)}
                  disabled={busy}
                  className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                />
                <label htmlFor="module-settings-published" className="text-sm text-slate-700">
                  Published to students
                </label>
              </div>

              <div className="mt-4 space-y-2">
                <RelativeScheduleBanner
                  scheduleMode={scheduleMode}
                  relativeAnchorAt={relativeScheduleAnchorAt}
                />
                <ScheduleDatetimeField
                  id={dateInputId}
                  label="Visible from (optional)"
                  relativeLabel="Visible after enrollment (optional)"
                  fixedHint="Leave empty to show when published. Uses your local timezone."
                  relativeHint="Leave empty to show when published. Offset is from each student’s enrollment."
                  value={visibleLocal}
                  onChange={setVisibleLocal}
                  disabled={busy}
                  scheduleMode={scheduleMode}
                  relativeAnchorAt={relativeScheduleAnchorAt}
                  defaultTime="00:00"
                  className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 disabled:cursor-not-allowed disabled:opacity-60"
                />
              </div>
              <button type="submit" className="sr-only" tabIndex={-1}>
                Save
              </button>
            </form>
          </div>

          {showRequirementsTab ? (
            <div
              id={requirementsPanelId}
              role="tabpanel"
              aria-labelledby={requirementsTabId}
              hidden={tab !== 'requirements'}
            >
              {requirementsSection}
            </div>
          ) : null}

          {errorMessage ? (
            <p className="mt-3 text-sm text-rose-700" role="alert">
              {errorMessage}
            </p>
          ) : null}
        </div>

        <div className="flex shrink-0 flex-wrap items-center gap-2 border-t border-slate-200 px-4 py-3">
          {onDelete && tab === 'general' ? (
            <button
              type="button"
              onClick={() => onDelete()}
              disabled={busy}
              className="rounded-xl px-3 py-2 text-sm font-medium text-rose-700 hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              Delete module
            </button>
          ) : null}
          <div className="ml-auto flex flex-wrap justify-end gap-2">
            <button
              type="button"
              onClick={() => onClose()}
              disabled={busy}
              className="rounded-xl px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => submitActiveTab()}
              disabled={saveDisabled}
              className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {saveLabel}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
