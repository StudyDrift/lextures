import { useCallback, useEffect, useId, useMemo, useState } from 'react'
import { Sparkles } from 'lucide-react'
import { Dialog } from '../../components/ui/dialog'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { DatePicker } from '../../components/ui/date-picker'
import { Textarea } from '../../components/ui/textarea'
import { Checkbox } from '../../components/ui/checkbox'
import { Tabs, TabList, Tab, TabPanel } from '../../components/ui/tabs'
import { SegmentedControl } from '../../components/ui/segmented-control'
import { formatAbsolute } from '../../lib/format-datetime'
import {
  bulkPatchCourseStructureItemDueAt,
  postAdjustDatesWithAi,
  type CourseStructureItem,
} from '../../lib/courses-api'
import { toast, toastMutationError } from '../../lib/lms-toast'
import {
  collectDateableItems,
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
type WorkTab = 'manual' | 'ai'

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
  const dateableItems = useMemo(() => collectDateableItems(structureItems), [structureItems])
  const undatedCount = useMemo(
    () => dateableItems.filter((it) => !it.dueAt).length,
    [dateableItems],
  )
  const isRelative = scheduleMode === 'relative'
  const canRunAi = aiConfigured && dateableItems.length > 0
  const preferAiTab = datedItems.length === 0 && dateableItems.length > 0

  const [workTab, setWorkTab] = useState<WorkTab>('manual')
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
    setWorkTab(preferAiTab ? 'ai' : 'manual')
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
  }, [open, preferAiTab])

  const applyPreview = useCallback((rows: DateChangePreview[]) => {
    setPreview(rows)
    setSelected(new Set(rows.map((r) => r.itemId)))
  }, [])

  const runManualPreview = useCallback(() => {
    setError(null)
    setAiReply(null)
    if (datedItems.length === 0) {
      setError('No items with due dates yet. Use the AI tab to propose an initial schedule.')
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
    if (!canRunAi || busy) return
    setBusy(true)
    setError(null)
    setAiReply(null)
    try {
      const res = await postAdjustDatesWithAi(courseCode, {
        instruction: aiInstruction.trim() || undefined,
      })
      setAiReply(res.reply)
      const rows = mergeAiProposals(
        dateableItems,
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
  }, [aiInstruction, applyPreview, busy, canRunAi, courseCode, dateableItems])

  const toggleSelected = useCallback((itemId: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(itemId)) next.delete(itemId)
      else next.add(itemId)
      return next
    })
  }, [])

  const selectAll = useCallback(() => {
    setSelected(new Set(preview.map((r) => r.itemId)))
  }, [preview])

  const selectNone = useCallback(() => {
    setSelected(new Set())
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

  const statusLine = useMemo(() => {
    if (dateableItems.length === 0) {
      return 'No assignments, quizzes, or pages yet.'
    }
    if (datedItems.length === 0) {
      return `${dateableItems.length} undated · set an initial schedule with AI`
    }
    if (undatedCount > 0) {
      return `${datedItems.length} dated · ${undatedCount} undated`
    }
    return `${datedItems.length} item${datedItems.length === 1 ? '' : 's'} with due dates`
  }, [dateableItems.length, datedItems.length, undatedCount])

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Adjust Dates"
      description={
        isRelative
          ? 'Bulk-edit due dates. Relative courses store absolute instants and re-anchor per enrollment.'
          : 'Bulk-edit due dates for assignments, quizzes, and pages. Preview, then accept.'
      }
      size="xl"
      closeLabel="Close"
      panelClassName="sm:max-h-[min(88vh,42rem)]"
      bodyClassName="flex !flex-col !overflow-hidden !py-3"
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
            {selectedCount > 0
              ? `Accept ${selectedCount} change${selectedCount === 1 ? '' : 's'}`
              : 'Accept changes'}
          </Button>
        </>
      }
    >
      {/*
        Inner flex layout: tools stay compact at top; the preview list claims
        remaining height and scrolls independently so the dialog never grows
        past the viewport (Dialog already caps height + scrolls the body).
      */}
      <div className="flex min-h-0 flex-1 flex-col gap-3">
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          <span className="rounded-full border border-border-subtle bg-surface-sunken px-2.5 py-0.5 text-xs text-fg-muted">
            {statusLine}
          </span>
          {isRelative && relativeScheduleAnchorAt ? (
            <span className="rounded-full border border-border-subtle bg-surface-sunken px-2.5 py-0.5 text-xs text-fg-muted">
              Anchor {formatAbsolute(relativeScheduleAnchorAt)}
            </span>
          ) : null}
        </div>

        <Tabs value={workTab} onValueChange={(v) => setWorkTab(v as WorkTab)} className="shrink-0">
          <TabList aria-label="Adjust dates method" className="w-full">
            <Tab value="manual" className="flex-1 justify-center">
              Manual
            </Tab>
            <Tab value="ai" className="flex-1 justify-center">
              AI
            </Tab>
          </TabList>

          <TabPanel value="manual" className="space-y-3 !py-2">
            <SegmentedControl<ManualMode>
              label="Mode"
              size="sm"
              value={manualMode}
              onChange={setManualMode}
              options={[
                { value: 'shift', label: 'Shift by days' },
                { value: 'rebase', label: 'Move earliest to…' },
              ]}
            />
            <div className="flex flex-wrap items-end gap-2">
              {manualMode === 'shift' ? (
                <div className="min-w-[7.5rem] flex-1 sm:flex-none">
                  <label
                    htmlFor={`${baseId}-days`}
                    className="mb-1 block text-xs font-medium text-fg-muted"
                  >
                    Days (negative = earlier)
                  </label>
                  <Input
                    id={`${baseId}-days`}
                    type="number"
                    step={1}
                    value={dayDelta}
                    onChange={(e) => setDayDelta(e.target.value)}
                    className="w-full sm:w-28"
                  />
                </div>
              ) : (
                <div className="min-w-[10rem] flex-1 sm:flex-none">
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
              <Button
                type="button"
                variant="secondary"
                onClick={runManualPreview}
                disabled={datedItems.length === 0}
              >
                Preview
              </Button>
            </div>
            {datedItems.length === 0 ? (
              <p className="text-xs text-fg-muted">
                Manual shift needs existing due dates. Switch to AI to set an initial schedule.
              </p>
            ) : null}
          </TabPanel>

          <TabPanel value="ai" className="space-y-3 !py-2">
            <div>
              <label htmlFor={`${baseId}-ai`} className="mb-1 block text-xs font-medium text-fg-muted">
                Guidance (optional)
              </label>
              <Textarea
                id={`${baseId}-ai`}
                rows={2}
                value={aiInstruction}
                onChange={(e) => setAiInstruction(e.target.value)}
                placeholder={
                  datedItems.length === 0
                    ? 'e.g. 4 week course, end-of-day deadlines'
                    : 'e.g. Shift the term to start Sept 8 and keep spacing'
                }
                disabled={!aiConfigured || busy || dateableItems.length === 0}
              />
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                variant="primary"
                onClick={() => void runAiAdjust()}
                loading={busy}
                disabled={!canRunAi || busy}
              >
                <Sparkles className="me-1.5 h-4 w-4" aria-hidden />
                {datedItems.length === 0 && dateableItems.length > 0
                  ? 'Set dates with AI'
                  : 'Adjust with AI'}
              </Button>
              {!aiConfigured ? (
                <span className="text-xs text-fg-muted">AI is not configured.</span>
              ) : dateableItems.length === 0 ? (
                <span className="text-xs text-fg-muted">Add course content first.</span>
              ) : (
                <span className="text-xs text-fg-muted">Proposals appear below for review.</span>
              )}
            </div>
            {aiReply ? (
              <p className="rounded-lg border border-border-subtle bg-surface-sunken px-3 py-2 text-sm text-fg-muted">
                {aiReply}
              </p>
            ) : null}
          </TabPanel>
        </Tabs>

        {error ? (
          <p
            className="rounded-lg border border-border-default bg-danger-surface px-3 py-2 text-sm text-danger-fg"
            role="alert"
          >
            {error}
          </p>
        ) : null}

        <section className="flex min-h-[10rem] flex-1 flex-col overflow-hidden rounded-xl border border-border-default">
          <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-border-subtle bg-surface-sunken px-3 py-2">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
              {preview.length > 0
                ? `Preview · ${selectedCount} of ${preview.length} selected`
                : 'Preview'}
            </h3>
            {preview.length > 0 ? (
              <div className="flex gap-1">
                <Button type="button" size="sm" variant="ghost" onClick={selectAll}>
                  Select all
                </Button>
                <Button type="button" size="sm" variant="ghost" onClick={selectNone}>
                  Select none
                </Button>
              </div>
            ) : null}
          </div>

          {preview.length > 0 ? (
            <ul className="min-h-0 flex-1 divide-y divide-border-subtle overflow-y-auto overscroll-contain">
              {preview.map((row) => {
                const checked = selected.has(row.itemId)
                return (
                  <li key={row.itemId}>
                    <label className="flex cursor-pointer items-start gap-3 px-3 py-2.5 hover:bg-surface-sunken/60">
                      <Checkbox
                        checked={checked}
                        onChange={() => toggleSelected(row.itemId)}
                        aria-label={`Include ${row.title}`}
                        className="mt-0.5"
                      />
                      <span className="min-w-0 flex-1">
                        <span className="block truncate text-sm font-medium text-fg-default">
                          {row.title}
                        </span>
                        <span className="block truncate text-xs text-fg-muted">
                          {kindLabel(row.kind)}
                          {row.moduleTitle ? ` · ${row.moduleTitle}` : ''}
                        </span>
                      </span>
                      <span className="shrink-0 text-end text-xs leading-snug">
                        <span className="block text-fg-muted">
                          {row.fromDueAt ? formatAbsolute(row.fromDueAt) : '—'}
                        </span>
                        <span className="block font-medium text-fg-default">
                          → {formatAbsolute(row.toDueAt)}
                        </span>
                      </span>
                    </label>
                  </li>
                )
              })}
            </ul>
          ) : (
            <div className="flex flex-1 items-center justify-center px-4 py-8 text-center text-sm text-fg-muted">
              {datedItems.length === 0 && dateableItems.length > 0
                ? 'Use the AI tab to propose an initial schedule, then review it here.'
                : 'Preview manual changes or run AI to review proposed due dates here.'}
            </div>
          )}
        </section>
      </div>
    </Dialog>
  )
}
