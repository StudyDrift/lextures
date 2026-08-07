import { useCallback, useEffect, useId, useMemo, useState } from 'react'
import { Sparkles } from 'lucide-react'
import { Dialog } from '../../components/ui/dialog'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { DatePicker } from '../../components/ui/date-picker'
import { Textarea } from '../../components/ui/textarea'
import { Checkbox } from '../../components/ui/checkbox'
import { formatAbsolute } from '../../lib/format-datetime'
import {
  bulkPatchCourseStructureItemDueAt,
  postAdjustDatesWithAi,
  type CourseStructureItem,
} from '../../lib/courses-api'
import { toast, toastMutationError } from '../../lib/lms-toast'
import {
  collectDatedItems,
  kindLabel,
  mergeAiProposals,
  rebaseDueDates,
  shiftDueDatesByDays,
  type DateChangePreview,
} from './adjust-dates-logic'

type Props = {
  open: boolean
  onClose: () => void
  courseCode: string
  structureItems: CourseStructureItem[]
  scheduleMode?: string | null
  relativeScheduleAnchorAt?: string | null
  aiConfigured?: boolean
  onApplied: () => void | Promise<void>
}

type ManualMode = 'shift' | 'rebase'

export function AdjustDatesModal({
  open,
  onClose,
  courseCode,
  structureItems,
  scheduleMode,
  relativeScheduleAnchorAt,
  aiConfigured = false,
  onApplied,
}: Props) {
  const baseId = useId()
  const datedItems = useMemo(() => collectDatedItems(structureItems), [structureItems])
  const isRelative = scheduleMode === 'relative'

  const [manualMode, setManualMode] = useState<ManualMode>('shift')
  const [dayDelta, setDayDelta] = useState('7')
  const [newEarliestDate, setNewEarliestDate] = useState('')
  const [aiInstruction, setAiInstruction] = useState('')
  const [preview, setPreview] = useState<DateChangePreview[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [aiReply, setAiReply] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [applying, setApplying] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setManualMode('shift')
    setDayDelta('7')
    setNewEarliestDate('')
    setAiInstruction('')
    setPreview([])
    setSelected(new Set())
    setAiReply(null)
    setError(null)
    setBusy(false)
    setApplying(false)
  }, [open])

  const applyPreview = useCallback((rows: DateChangePreview[]) => {
    setPreview(rows)
    setSelected(new Set(rows.map((r) => r.itemId)))
  }, [])

  const runManualPreview = useCallback(() => {
    setError(null)
    setAiReply(null)
    if (datedItems.length === 0) {
      setError('No items with due dates in this course.')
      applyPreview([])
      return
    }
    if (manualMode === 'shift') {
      const n = Number.parseInt(dayDelta, 10)
      if (!Number.isFinite(n) || n === 0) {
        setError('Enter a non-zero whole number of days (negative moves dates earlier).')
        applyPreview([])
        return
      }
      applyPreview(shiftDueDatesByDays(datedItems, n))
      return
    }
    if (!newEarliestDate.trim()) {
      setError('Pick a new date for the earliest due date.')
      applyPreview([])
      return
    }
    // Keep time-of-day from the current earliest item.
    const earliest = datedItems[0]
    const from = new Date(earliest.dueAt)
    const [y, m, d] = newEarliestDate.split('-').map((x) => Number.parseInt(x, 10))
    if (!y || !m || !d) {
      setError('Invalid date.')
      applyPreview([])
      return
    }
    const rebased = new Date(from)
    rebased.setFullYear(y, m - 1, d)
    applyPreview(rebaseDueDates(datedItems, rebased.toISOString()))
  }, [applyPreview, datedItems, dayDelta, manualMode, newEarliestDate])

  const runAiAdjust = useCallback(async () => {
    if (!aiConfigured || busy) return
    setBusy(true)
    setError(null)
    setAiReply(null)
    try {
      const res = await postAdjustDatesWithAi(courseCode, {
        instruction: aiInstruction.trim() || undefined,
      })
      setAiReply(res.reply)
      const rows = mergeAiProposals(
        datedItems,
        res.proposals.map((p) => ({ itemId: p.itemId, dueAt: p.dueAt })),
      )
      if (rows.length === 0) {
        setError(res.reply || 'AI did not propose any date changes.')
        applyPreview([])
      } else {
        applyPreview(rows)
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Adjust with AI failed.'
      setError(msg)
      toastMutationError(msg)
    } finally {
      setBusy(false)
    }
  }, [aiConfigured, aiInstruction, applyPreview, busy, courseCode, datedItems])

  const toggleSelected = useCallback((itemId: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(itemId)) next.delete(itemId)
      else next.add(itemId)
      return next
    })
  }, [])

  const selectedCount = useMemo(
    () => preview.filter((r) => selected.has(r.itemId)).length,
    [preview, selected],
  )

  const acceptChanges = useCallback(async () => {
    const updates = preview
      .filter((r) => selected.has(r.itemId))
      .map((r) => ({ itemId: r.itemId, dueAt: r.toDueAt }))
    if (updates.length === 0) {
      setError('Select at least one change to accept.')
      return
    }
    setApplying(true)
    setError(null)
    try {
      const result = await bulkPatchCourseStructureItemDueAt(courseCode, { updates })
      toast(
        `Updated ${result.updated} due date${result.updated === 1 ? '' : 's'}${
          result.failed > 0 ? ` (${result.failed} failed)` : ''
        }`,
      )
      await onApplied()
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Could not apply date changes.'
      setError(msg)
      toastMutationError(msg)
    } finally {
      setApplying(false)
    }
  }, [courseCode, onApplied, onClose, preview, selected])

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Adjust Dates"
      description={
        isRelative
          ? 'Bulk-adjust due dates for this course. Dates are stored as absolute instants and shift relative to each learner’s enrollment when the course uses relative scheduling.'
          : 'Bulk-adjust due dates for assignments, quizzes, and content pages. Preview changes, then accept.'
      }
      size="xl"
      closeLabel="Close"
      footer={
        <>
          <Button type="button" variant="secondary" onClick={onClose} disabled={applying}>
            Cancel
          </Button>
          <Button
            type="button"
            variant="primary"
            onClick={() => void acceptChanges()}
            loading={applying}
            disabled={selectedCount === 0 || applying}
          >
            {selectedCount > 0 ? `Accept ${selectedCount} change${selectedCount === 1 ? '' : 's'}` : 'Accept changes'}
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {isRelative && relativeScheduleAnchorAt ? (
          <p className="rounded-lg border border-border-subtle bg-surface-sunken px-3 py-2 text-xs text-fg-muted">
            Relative schedule mode · anchor{' '}
            <span className="font-medium text-fg-default">
              {formatAbsolute(relativeScheduleAnchorAt)}
            </span>
          </p>
        ) : null}

        <p className="text-sm text-fg-muted">
          {datedItems.length === 0
            ? 'No dated items yet. Add due dates to assignments, quizzes, or pages first.'
            : `${datedItems.length} item${datedItems.length === 1 ? '' : 's'} with due dates.`}
        </p>

        <div className="rounded-xl border border-border-default p-3">
          <div className="flex flex-wrap gap-2" role="group" aria-label="Manual adjust mode">
            <Button
              type="button"
              size="sm"
              variant={manualMode === 'shift' ? 'primary' : 'secondary'}
              onClick={() => setManualMode('shift')}
            >
              Shift by days
            </Button>
            <Button
              type="button"
              size="sm"
              variant={manualMode === 'rebase' ? 'primary' : 'secondary'}
              onClick={() => setManualMode('rebase')}
            >
              Move earliest to…
            </Button>
          </div>

          <div className="mt-3 flex flex-wrap items-end gap-3">
            {manualMode === 'shift' ? (
              <div className="min-w-[8rem]">
                <label htmlFor={`${baseId}-days`} className="mb-1 block text-xs font-medium text-fg-muted">
                  Days (negative = earlier)
                </label>
                <Input
                  id={`${baseId}-days`}
                  type="number"
                  step={1}
                  value={dayDelta}
                  onChange={(e) => setDayDelta(e.target.value)}
                  className="w-28"
                />
              </div>
            ) : (
              <div className="min-w-[10rem]">
                <label
                  htmlFor={`${baseId}-earliest`}
                  className="mb-1 block text-xs font-medium text-fg-muted"
                >
                  New earliest due date
                </label>
                <DatePicker
                  id={`${baseId}-earliest`}
                  type="date"
                  value={newEarliestDate}
                  onChange={(e) => setNewEarliestDate(e.target.value)}
                />
              </div>
            )}
            <Button type="button" variant="secondary" onClick={runManualPreview} disabled={datedItems.length === 0}>
              Preview
            </Button>
          </div>
        </div>

        <div className="rounded-xl border border-border-default p-3">
          <label htmlFor={`${baseId}-ai`} className="mb-1 block text-xs font-medium text-fg-muted">
            Optional AI guidance
          </label>
          <Textarea
            id={`${baseId}-ai`}
            rows={2}
            value={aiInstruction}
            onChange={(e) => setAiInstruction(e.target.value)}
            placeholder="e.g. Shift the whole term to start the week of Sept 8 and keep spacing."
            disabled={!aiConfigured || busy}
          />
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <Button
              type="button"
              variant="primary"
              onClick={() => void runAiAdjust()}
              loading={busy}
              disabled={!aiConfigured || datedItems.length === 0 || busy}
            >
              <Sparkles className="me-1.5 h-4 w-4" aria-hidden />
              Adjust with AI
            </Button>
            {!aiConfigured ? (
              <span className="text-xs text-fg-muted">AI is not configured for this environment.</span>
            ) : null}
          </div>
          {aiReply ? <p className="mt-2 text-sm text-fg-muted">{aiReply}</p> : null}
        </div>

        {error ? (
          <p className="rounded-lg border border-border-default bg-danger-surface px-3 py-2 text-sm text-danger-fg" role="alert">
            {error}
          </p>
        ) : null}

        {preview.length > 0 ? (
          <div className="max-h-72 overflow-auto rounded-xl border border-border-default">
            <table className="w-full text-start text-sm">
              <thead className="sticky top-0 bg-surface-sunken text-xs text-fg-muted">
                <tr>
                  <th className="px-2 py-2 font-medium">Include</th>
                  <th className="px-2 py-2 font-medium">Item</th>
                  <th className="px-2 py-2 font-medium">From</th>
                  <th className="px-2 py-2 font-medium">To</th>
                </tr>
              </thead>
              <tbody>
                {preview.map((row) => (
                  <tr key={row.itemId} className="border-t border-border-subtle">
                    <td className="px-2 py-2 align-top">
                      <Checkbox
                        checked={selected.has(row.itemId)}
                        onChange={() => toggleSelected(row.itemId)}
                        aria-label={`Include ${row.title}`}
                      />
                    </td>
                    <td className="px-2 py-2 align-top">
                      <div className="font-medium text-fg-default">{row.title}</div>
                      <div className="text-xs text-fg-muted">
                        {kindLabel(row.kind)}
                        {row.moduleTitle ? ` · ${row.moduleTitle}` : ''}
                      </div>
                    </td>
                    <td className="px-2 py-2 align-top text-fg-muted">{formatAbsolute(row.fromDueAt)}</td>
                    <td className="px-2 py-2 align-top font-medium text-fg-default">
                      {formatAbsolute(row.toDueAt)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="text-sm text-fg-muted">
            Preview manual changes or run Adjust with AI to review proposed due dates before accepting.
          </p>
        )}
      </div>
    </Dialog>
  )
}
