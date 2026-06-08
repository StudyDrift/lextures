import { useEffect, useId, useState } from 'react'
import { createPortal } from 'react-dom'
import type { GradebookColumn } from './gradebook-grid-types'
import type { IncompleteGradeRecord } from '../../../lib/incomplete-grades-api'
import {
  fetchIncompleteGrade,
  grantIncompleteGrade,
  resolveIncompleteGrade,
} from '../../../lib/incomplete-grades-api'

type IncompleteGradeModalProps = {
  open: boolean
  courseCode: string
  enrollmentId: string
  studentName: string
  assignmentColumns: GradebookColumn[]
  existingRecord?: IncompleteGradeRecord | null
  onClose: () => void
  onSaved: () => void
}

function defaultDeadline(): string {
  const d = new Date()
  d.setDate(d.getDate() + 90)
  return d.toISOString().slice(0, 10)
}

function daysUntil(dateStr: string): number {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const end = new Date(`${dateStr}T00:00:00`)
  return Math.ceil((end.getTime() - today.getTime()) / (1000 * 60 * 60 * 24))
}

export function IncompleteGradeModal({
  open,
  courseCode,
  enrollmentId,
  studentName,
  assignmentColumns,
  existingRecord,
  onClose,
  onSaved,
}: IncompleteGradeModalProps) {
  const titleId = useId()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [record, setRecord] = useState<IncompleteGradeRecord | null>(existingRecord ?? null)
  const [extensionDeadline, setExtensionDeadline] = useState(defaultDeadline())
  const [selectedItems, setSelectedItems] = useState<string[]>([])
  const [notes, setNotes] = useState('')
  const [resolvedGrade, setResolvedGrade] = useState('')

  const assignmentOptions = assignmentColumns.filter((c) => c.kind === 'assignment')

  useEffect(() => {
    if (!open) return
    setError(null)
    setRecord(existingRecord ?? null)
    setExtensionDeadline(existingRecord?.extensionDeadline ?? defaultDeadline())
    setSelectedItems(existingRecord?.outstandingItemIds ?? [])
    setNotes(existingRecord?.notes ?? '')
    setResolvedGrade('')
    if (existingRecord != null) return
    setLoading(true)
    void fetchIncompleteGrade(courseCode, enrollmentId)
      .then((r) => {
        if (r) {
          setRecord(r)
          setExtensionDeadline(r.extensionDeadline)
          setSelectedItems(r.outstandingItemIds)
          setNotes(r.notes ?? '')
        }
      })
      .catch((e: unknown) => {
        setError(e instanceof Error ? e.message : 'Failed to load incomplete record.')
      })
      .finally(() => setLoading(false))
  }, [open, courseCode, enrollmentId, existingRecord])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  const isOpen = record?.status === 'open'

  async function handleGrant(e: React.FormEvent) {
    e.preventDefault()
    if (selectedItems.length === 0) {
      setError('Select at least one outstanding assignment.')
      return
    }
    setSaving(true)
    setError(null)
    try {
      await grantIncompleteGrade(courseCode, enrollmentId, {
        extensionDeadline,
        outstandingItemIds: selectedItems,
        notes: notes.trim() || undefined,
      })
      onSaved()
      onClose()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to grant incomplete.')
    } finally {
      setSaving(false)
    }
  }

  async function handleResolve(e: React.FormEvent) {
    e.preventDefault()
    if (!resolvedGrade.trim()) {
      setError('Enter the final resolved grade.')
      return
    }
    setSaving(true)
    setError(null)
    try {
      await resolveIncompleteGrade(courseCode, enrollmentId, resolvedGrade.trim())
      onSaved()
      onClose()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to resolve incomplete.')
    } finally {
      setSaving(false)
    }
  }

  function toggleItem(id: string) {
    setSelectedItems((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  const panel = (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" role="presentation">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="w-full max-w-lg rounded-xl border border-slate-200 bg-white p-5 shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
      >
        <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
          {isOpen ? 'Incomplete grade' : 'Grant Incomplete'}
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">{studentName}</p>

        {loading ? (
          <p className="mt-4 text-sm text-slate-500">Loading…</p>
        ) : (
          <>
            {isOpen && record ? (
              <div className="mt-4 space-y-3 text-sm">
                <p>
                  <span className="font-medium">Extension deadline:</span>{' '}
                  {record.extensionDeadline}
                  <span className="ms-2 text-violet-700 dark:text-violet-300">
                    ({daysUntil(record.extensionDeadline)} days left)
                  </span>
                </p>
                {record.outstandingItemIds.length > 0 ? (
                  <p className="text-slate-600 dark:text-neutral-400">
                    Outstanding assignments selected when granted.
                  </p>
                ) : null}
                <form onSubmit={handleResolve} className="space-y-3 pt-2">
                  <div>
                    <label htmlFor="resolved-grade" className="block text-sm font-medium mb-1">
                      Final resolved grade
                    </label>
                    <input
                      id="resolved-grade"
                      type="text"
                      value={resolvedGrade}
                      onChange={(e) => setResolvedGrade(e.target.value)}
                      className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                      placeholder="e.g. B+"
                    />
                  </div>
                  <div className="flex justify-end gap-2 pt-2">
                    <button
                      type="button"
                      onClick={onClose}
                      className="rounded-lg px-3 py-2 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={saving}
                      className="rounded-lg bg-violet-600 px-3 py-2 text-sm font-medium text-white hover:bg-violet-700 disabled:opacity-60"
                    >
                      {saving ? 'Saving…' : 'Resolve Incomplete'}
                    </button>
                  </div>
                </form>
              </div>
            ) : (
              <form onSubmit={handleGrant} className="mt-4 space-y-4">
                <div>
                  <label htmlFor="extension-deadline" className="block text-sm font-medium mb-1">
                    Extension deadline
                  </label>
                  <input
                    id="extension-deadline"
                    type="date"
                    required
                    value={extensionDeadline}
                    min={new Date().toISOString().slice(0, 10)}
                    onChange={(e) => setExtensionDeadline(e.target.value)}
                    className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                  />
                </div>
                <fieldset>
                  <legend className="text-sm font-medium mb-2">Outstanding assignments</legend>
                  <div className="max-h-40 overflow-y-auto space-y-1 rounded-lg border border-slate-200 p-2 dark:border-neutral-700">
                    {assignmentOptions.length === 0 ? (
                      <p className="text-sm text-slate-500">No assignment columns in this gradebook.</p>
                    ) : (
                      assignmentOptions.map((col) => (
                        <label key={col.id} className="flex items-center gap-2 text-sm">
                          <input
                            type="checkbox"
                            checked={selectedItems.includes(col.id)}
                            onChange={() => toggleItem(col.id)}
                          />
                          {col.title}
                        </label>
                      ))
                    )}
                  </div>
                </fieldset>
                <div>
                  <label htmlFor="incomplete-notes" className="block text-sm font-medium mb-1">
                    Notes (optional)
                  </label>
                  <textarea
                    id="incomplete-notes"
                    rows={2}
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                  />
                </div>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={onClose}
                    className="rounded-lg px-3 py-2 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={saving}
                    className="rounded-lg bg-violet-600 px-3 py-2 text-sm font-medium text-white hover:bg-violet-700 disabled:opacity-60"
                  >
                    {saving ? 'Saving…' : 'Grant Incomplete'}
                  </button>
                </div>
              </form>
            )}
          </>
        )}

        {error ? (
          <div role="alert" className="mt-3 text-sm text-red-600">
            {error}
          </div>
        ) : null}
      </div>
    </div>
  )

  return createPortal(panel, document.body)
}
