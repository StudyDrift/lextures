import { type FormEvent, type ReactNode, useCallback, useEffect, useId, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { Check, Copy, KeyRound, X } from 'lucide-react'
import { authorizedFetch } from '../../lib/api'
import type { CoursePublic } from '../../lib/courses-api'
import { readApiErrorMessage } from '../../lib/errors'
import { toastMutationError } from '../../lib/lms-toast'

type ScopeDef = {
  id: string
  label: string
  description: string
  group: string
}

type CourseLimitMode = 'all' | 'specific'

function courseCodeLabel(c: CoursePublic): string {
  return c.courseCode ?? ''
}

function courseTitleLabel(c: CoursePublic): string {
  const title = typeof c.title === 'string' ? c.title.trim() : ''
  return title || 'Untitled course'
}

function compareCoursesByCode(a: CoursePublic, b: CoursePublic): number {
  return courseCodeLabel(a).localeCompare(courseCodeLabel(b), undefined, { sensitivity: 'base' })
}

function courseMatchesSearch(c: CoursePublic, q: string): boolean {
  if (!q) return true
  const haystack = [courseCodeLabel(c), courseTitleLabel(c), c.id ?? ''].join(' ').toLowerCase()
  return haystack.includes(q)
}

export type CreateAccessKeyResult = {
  token: string
  label: string
}

type CreateAccessKeyModalProps = {
  open: boolean
  scopes: ScopeDef[]
  onClose: () => void
  onCreated: (result: CreateAccessKeyResult) => void
}

function ModalSection({
  step,
  title,
  description,
  children,
}: {
  step: number
  title: string
  description: string
  children: ReactNode
}) {
  return (
    <section className="rounded-xl border border-slate-200 bg-slate-50/80 p-4 dark:border-neutral-700 dark:bg-neutral-800/40">
      <div className="flex gap-3">
        <span
          className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-indigo-600 text-xs font-bold text-white dark:bg-neutral-100 dark:text-neutral-950"
          aria-hidden
        >
          {step}
        </span>
        <div className="min-w-0 flex-1">
          <h4 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">{title}</h4>
          <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">{description}</p>
          <div className="mt-3">{children}</div>
        </div>
      </div>
    </section>
  )
}

export function CreateAccessKeyModal({ open, scopes, onClose, onCreated }: CreateAccessKeyModalProps) {
  const labelId = useId()
  const courseSearchId = useId()
  const titleId = useId()
  const descId = useId()

  const [creating, setCreating] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [label, setLabel] = useState('')
  const [selectedScopes, setSelectedScopes] = useState<string[]>(['mcp:connect', 'courses:read'])
  const [expiresAt, setExpiresAt] = useState('')
  const [courseLimitMode, setCourseLimitMode] = useState<CourseLimitMode>('all')
  const [selectedCourseIds, setSelectedCourseIds] = useState<string[]>([])
  const [availableCourses, setAvailableCourses] = useState<CoursePublic[]>([])
  const [coursesLoading, setCoursesLoading] = useState(false)
  const [courseSearch, setCourseSearch] = useState('')

  const scopeGroups = useMemo(() => {
    const map = new Map<string, ScopeDef[]>()
    for (const s of scopes) {
      const list = map.get(s.group) ?? []
      list.push(s)
      map.set(s.group, list)
    }
    return [...map.entries()].sort(([a], [b]) => (a ?? '').localeCompare(b ?? ''))
  }, [scopes])

  const filteredCourses = useMemo(() => {
    if (courseLimitMode !== 'specific') return []
    const q = courseSearch.trim().toLowerCase()
    const sorted = [...availableCourses].sort(compareCoursesByCode)
    if (!q) return sorted
    return sorted.filter((c) => courseMatchesSearch(c, q))
  }, [availableCourses, courseLimitMode, courseSearch])

  const resetForm = useCallback(() => {
    setLabel('')
    setExpiresAt('')
    setCourseLimitMode('all')
    setSelectedCourseIds([])
    setCourseSearch('')
    setSelectedScopes(['mcp:connect', 'courses:read'])
    setFormError(null)
  }, [])

  const loadCourses = useCallback(async () => {
    setCoursesLoading(true)
    try {
      const res = await authorizedFetch('/api/v1/courses')
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(readApiErrorMessage(raw))
      setAvailableCourses(
        ((raw as { courses?: CoursePublic[] }).courses ?? []).filter((c) => Boolean(c?.id)),
      )
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not load courses.')
    } finally {
      setCoursesLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!open) return
    resetForm()
    void loadCourses()
  }, [open, loadCourses, resetForm])

  useEffect(() => {
    if (!open || creating) return
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open, creating, onClose])

  function toggleScope(id: string) {
    setSelectedScopes((prev) =>
      prev.includes(id) ? prev.filter((s) => s !== id) : [...prev, id],
    )
  }

  function toggleCourse(id: string) {
    setSelectedCourseIds((prev) =>
      prev.includes(id) ? prev.filter((c) => c !== id) : [...prev, id],
    )
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!label.trim() || selectedScopes.length === 0) return
    if (courseLimitMode === 'specific' && selectedCourseIds.length === 0) {
      setFormError('Pick at least one course, or switch to “All my courses”.')
      return
    }
    setCreating(true)
    setFormError(null)
    try {
      const body: { label: string; scopes: string[]; courseIds?: string[]; expiresAt?: string } = {
        label: label.trim(),
        scopes: selectedScopes,
      }
      if (courseLimitMode === 'specific') {
        body.courseIds = selectedCourseIds
      }
      if (expiresAt.trim()) {
        body.expiresAt = new Date(expiresAt).toISOString()
      }
      const res = await authorizedFetch('/api/v1/me/access-keys', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setFormError(readApiErrorMessage(raw))
        return
      }
      const created = raw as { token?: string; label?: string }
      if (!created.token) {
        setFormError('Key was created but the secret was missing from the response.')
        return
      }
      onCreated({ token: created.token, label: created.label ?? label.trim() })
      onClose()
    } catch {
      setFormError('Could not create access key.')
    } finally {
      setCreating(false)
    }
  }

  const canSubmit =
    label.trim().length > 0 &&
    selectedScopes.length > 0 &&
    (courseLimitMode === 'all' || selectedCourseIds.length > 0)

  if (!open) return null

  return createPortal(
    <div className="fixed inset-0 z-[400] flex items-end justify-center p-4 sm:items-center" role="presentation">
      <button
        type="button"
        aria-label="Close dialog"
        disabled={creating}
        className="absolute inset-0 cursor-default border-0 bg-black/45 p-0 disabled:cursor-not-allowed"
        onClick={() => {
          if (!creating) onClose()
        }}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={descId}
        className="relative z-10 flex max-h-[min(92vh,720px)] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <div className="shrink-0 border-b border-slate-200 px-5 py-4 dark:border-neutral-700">
          <div className="flex items-start justify-between gap-3">
            <div className="flex items-start gap-3">
              <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-indigo-50 text-indigo-700 dark:bg-neutral-800 dark:text-neutral-100">
                <KeyRound className="h-5 w-5" aria-hidden />
              </span>
              <div>
                <h2 id={titleId} className="text-lg font-semibold text-slate-950 dark:text-neutral-100">
                  Create access key
                </h2>
                <p id={descId} className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
                  Set permissions and course limits, then copy the secret once — it won&apos;t be shown again.
                </p>
              </div>
            </div>
            <button
              type="button"
              onClick={onClose}
              disabled={creating}
              className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
              aria-label="Close"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>

        <form
          onSubmit={(e) => void onSubmit(e)}
          className="flex min-h-0 flex-1 flex-col overflow-hidden"
        >
          <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-5 py-4">
            <div className="space-y-4">
              {formError && (
              <p
                className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200"
                role="alert"
              >
                {formError}
              </p>
            )}

            <ModalSection
              step={1}
              title="Name this key"
              description="Use something you'll recognize later, like which tool or workflow it powers."
            >
              <input
                id={labelId}
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                placeholder="e.g. Cursor agent — CS101 grades"
                autoFocus
                className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
                required
              />
            </ModalSection>

            <ModalSection
              step={2}
              title="Choose permissions"
              description="Grant only what the tool needs. You can always create another key with different scopes."
            >
              <div className="space-y-4">
                {scopeGroups.map(([group, items]) => (
                  <fieldset key={group} className="min-w-0">
                    <legend className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                      {group}
                    </legend>
                    <ul className="mt-2 space-y-2">
                      {items.map((s) => {
                        const checked = selectedScopes.includes(s.id)
                        return (
                          <li key={s.id}>
                            <label className="flex cursor-pointer gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2.5 transition-[background-color,color,border-color] hover:border-indigo-200 dark:border-neutral-600 dark:bg-neutral-900 dark:hover:border-indigo-800/60">
                              <input
                                type="checkbox"
                                checked={checked}
                                onChange={() => toggleScope(s.id)}
                                className="mt-0.5 shrink-0"
                              />
                              <span className="min-w-0">
                                <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                                  {s.label}
                                </span>
                                <span className="block text-xs text-slate-500 dark:text-neutral-400">
                                  {s.description}
                                </span>
                              </span>
                            </label>
                          </li>
                        )
                      })}
                    </ul>
                  </fieldset>
                ))}
              </div>
              <p className="mt-3 text-xs text-slate-500 dark:text-neutral-400">
                {selectedScopes.length} permission{selectedScopes.length === 1 ? '' : 's'} selected
              </p>
            </ModalSection>

            <ModalSection
              step={3}
              title="Limit to courses"
              description="Keep the key on all courses you can access, or restrict it to a subset."
            >
              <div role="radiogroup" aria-label="Course access limit" className="grid gap-2 sm:grid-cols-2">
                <button
                  type="button"
                  role="radio"
                  aria-checked={courseLimitMode === 'all'}
                  onClick={() => setCourseLimitMode('all')}
                  className={`cursor-pointer rounded-xl border px-3 py-3 text-start transition-[background-color,color,border-color] ${
                    courseLimitMode === 'all'
                      ? 'border-indigo-500 bg-indigo-50 ring-1 ring-indigo-500/30 dark:border-indigo-400 dark:bg-indigo-950/40'
                      : 'border-slate-200 bg-white hover:border-slate-300 dark:border-neutral-600 dark:bg-neutral-900'
                  }`}
                >
                  <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                    All my courses
                  </span>
                  <span className="mt-0.5 block text-xs text-slate-500 dark:text-neutral-400">
                    Any course you can open in the app
                  </span>
                </button>
                <button
                  type="button"
                  role="radio"
                  aria-checked={courseLimitMode === 'specific'}
                  onClick={() => setCourseLimitMode('specific')}
                  className={`cursor-pointer rounded-xl border px-3 py-3 text-start transition-[background-color,color,border-color] ${
                    courseLimitMode === 'specific'
                      ? 'border-indigo-500 bg-indigo-50 ring-1 ring-indigo-500/30 dark:border-indigo-400 dark:bg-indigo-950/40'
                      : 'border-slate-200 bg-white hover:border-slate-300 dark:border-neutral-600 dark:bg-neutral-900'
                  }`}
                >
                  <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                    Selected courses
                  </span>
                  <span className="mt-0.5 block text-xs text-slate-500 dark:text-neutral-400">
                    Pick one or more from a list
                  </span>
                </button>
              </div>

              {courseLimitMode === 'specific' && (
                <div className="mt-3 space-y-2">
                  <input
                    id={courseSearchId}
                    value={courseSearch}
                    onChange={(e) => setCourseSearch(e.target.value)}
                    placeholder="Search by course code or title…"
                    className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
                  />
                  {coursesLoading ? (
                    <p className="text-sm text-slate-500 dark:text-neutral-400">Loading courses…</p>
                  ) : filteredCourses.length === 0 ? (
                    <p className="text-sm text-slate-500 dark:text-neutral-400">No matching courses.</p>
                  ) : (
                    <ul className="max-h-40 space-y-1 overflow-y-auto rounded-xl border border-slate-200 bg-white p-1 dark:border-neutral-600 dark:bg-neutral-950">
                      {filteredCourses.map((c) => {
                        const checked = selectedCourseIds.includes(c.id)
                        return (
                          <li key={c.id}>
                            <label className="flex cursor-pointer items-start gap-2 rounded-lg px-2 py-1.5 hover:bg-slate-50 dark:hover:bg-neutral-800">
                              <input
                                type="checkbox"
                                checked={checked}
                                onChange={() => toggleCourse(c.id)}
                                className="mt-0.5"
                              />
                              <span className="min-w-0">
                                <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                                  {courseCodeLabel(c) || 'No course code'}
                                </span>
                                <span className="block truncate text-xs text-slate-500 dark:text-neutral-400">
                                  {courseTitleLabel(c)}
                                </span>
                              </span>
                            </label>
                          </li>
                        )
                      })}
                    </ul>
                  )}
                  <p className="text-xs text-slate-500 dark:text-neutral-400">
                    {selectedCourseIds.length} course{selectedCourseIds.length === 1 ? '' : 's'} selected
                  </p>
                </div>
              )}
            </ModalSection>

            <ModalSection
              step={4}
              title="Expiration (optional)"
              description="Leave blank if this key should stay valid until you revoke it."
            >
              <input
                type="datetime-local"
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
                className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
              />
            </ModalSection>
            </div>
          </div>

          <div className="shrink-0 flex flex-wrap justify-end gap-2 border-t border-slate-200 px-5 py-4 dark:border-neutral-700">
            <button
              type="button"
              onClick={onClose}
              disabled={creating}
              className="rounded-xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 disabled:opacity-50 dark:border-neutral-600 dark:text-neutral-200"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={creating || !canSubmit}
              className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-950"
            >
              {creating ? 'Creating…' : 'Create key'}
            </button>
          </div>
        </form>
      </div>
    </div>,
    document.body,
  )
}

type AccessKeyCreatedModalProps = {
  open: boolean
  token: string | null
  label: string | null
  onClose: () => void
}

export function AccessKeyCreatedModal({ open, token, label, onClose }: AccessKeyCreatedModalProps) {
  const titleId = useId()
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!open) {
      queueMicrotask(() => setCopied(false))
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open, onClose])

  async function copyToken() {
    if (!token) return
    try {
      await navigator.clipboard.writeText(token)
      setCopied(true)
    } catch {
      toastMutationError('Could not copy to clipboard.')
    }
  }

  if (!open || !token) return null

  return (
    <div className="fixed inset-0 z-[410] flex items-end justify-center p-4 sm:items-center" role="presentation">
      <button
        type="button"
        aria-label="Close dialog"
        className="absolute inset-0 cursor-default border-0 bg-black/45 p-0"
        onClick={onClose}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="relative z-10 w-full max-w-xl overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
      >
        <div className="border-b border-slate-200 px-5 py-4 dark:border-neutral-700">
          <div className="flex items-start justify-between gap-3">
            <div>
              <h2 id={titleId} className="text-lg font-semibold text-slate-950 dark:text-neutral-100">
                Copy your access key
              </h2>
              {label ? (
                <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">{label}</p>
              ) : null}
            </div>
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
              aria-label="Close"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>

        <div className="space-y-4 px-5 py-4">
          <p
            className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100"
            role="status"
            aria-live="polite"
          >
            Token generated — copy now. This is the only time we&apos;ll show the full key.
          </p>
          <code className="block break-all rounded-xl border border-slate-200 bg-slate-50 px-3 py-3 font-mono text-xs text-slate-800 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100">
            {token}
          </code>
          <button
            type="button"
            onClick={() => void copyToken()}
            className="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white"
          >
            {copied ? <Check className="h-4 w-4" aria-hidden /> : <Copy className="h-4 w-4" aria-hidden />}
            {copied ? 'Copied' : 'Copy access key'}
          </button>
        </div>

        <div className="border-t border-slate-200 px-5 py-4 dark:border-neutral-700">
          <button
            type="button"
            onClick={onClose}
            className="w-full rounded-xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 dark:border-neutral-600 dark:text-neutral-200"
          >
            Done — I&apos;ve saved it
          </button>
        </div>
      </div>
    </div>
  )
}
