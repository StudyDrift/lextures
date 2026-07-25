import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Check,
  Loader2,
  Plus,
  RefreshCw,
  Sparkles,
  X,
} from 'lucide-react'
import {
  approveAdaptiveContentVariant,
  bulkReviewAdaptiveContentVariants,
  createAdaptiveContentUnit,
  editApproveAdaptiveContentVariant,
  fetchAdaptiveContentBudget,
  fetchAdaptiveContentEffectiveness,
  fetchAdaptiveContentKeyTerms,
  fetchAdaptiveContentReviewQueue,
  fetchAdaptiveContentSettings,
  fetchAdaptiveContentUnitEffectiveness,
  fetchAdaptiveContentUnits,
  fetchAdaptiveContentVariants,
  fetchCourseStructure,
  patchAdaptiveContentUnit,
  previewAdaptiveContentVariant,
  prewarmAdaptiveContentUnit,
  putAdaptiveContentKeyTerms,
  putAdaptiveContentSettings,
  refreshAdaptiveContentEffectiveness,
  rejectAdaptiveContentVariant,
  revokeAdaptiveContentVariant,
  type AdaptiveContentBudget,
  type AdaptiveContentEffectiveness,
  type AdaptiveContentKeyTerm,
  type AdaptiveContentPreview,
  type AdaptiveContentSettings,
  type AdaptiveContentUnit,
  type AdaptiveContentVariant,
  type CourseStructureItem,
} from '../../../lib/courses-api'
import { useConfirm } from '../../use-confirm'
import { EffectivenessChip, EffectivenessSummaryTable } from './effectiveness-chip'
import { VariantDiff } from './variant-diff'

const ALL_AXES = [
  { id: 'emphasis', label: 'Emphasis mode' },
  { id: 'scaffolding', label: 'Scaffolding' },
  { id: 'reading_level', label: 'Reading level' },
  { id: 'misconception', label: 'Misconception targeting' },
  { id: 'modality', label: 'Modality hint' },
] as const

const EMPHASIS_MODES = ['introduce', 'reinforce', 'compress', 'remediate'] as const

type Props = {
  courseCode: string
  adaptiveContentEnabled: boolean
  /** When false, hide unit config mutations (TA review-only). Defaults true. */
  canConfigure?: boolean
}

type WorkspaceTab = 'units' | 'queue'

function statusBadgeClass(status: string): string {
  switch (status) {
    case 'active':
    case 'approved':
    case 'auto_served':
      return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-200'
    case 'pending_review':
    case 'draft':
      return 'bg-amber-100 text-amber-900 dark:bg-amber-950/40 dark:text-amber-200'
    case 'rejected':
    case 'paused':
      return 'bg-rose-100 text-rose-800 dark:bg-rose-950/40 dark:text-rose-200'
    case 'superseded':
    case 'archived':
      return 'bg-slate-100 text-slate-600 dark:bg-neutral-800 dark:text-neutral-300'
    default:
      return 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-200'
  }
}

function fidelityLabel(score: number | null | undefined): string {
  if (score == null || Number.isNaN(score)) return '—'
  return `${Math.round(score * 100)}%`
}

export function AdaptiveContentWorkspace({
  courseCode,
  adaptiveContentEnabled,
  canConfigure = true,
}: Props) {
  const { confirm, ConfirmDialogHost } = useConfirm()
  const [tab, setTab] = useState<WorkspaceTab>('units')
  const [units, setUnits] = useState<AdaptiveContentUnit[]>([])
  const [structure, setStructure] = useState<CourseStructureItem[]>([])
  const [budget, setBudget] = useState<AdaptiveContentBudget | null>(null)
  const [settings, setSettings] = useState<AdaptiveContentSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [statusLive, setStatusLive] = useState('')

  const [selectedUnitId, setSelectedUnitId] = useState<string | null>(null)
  const [editorOpen, setEditorOpen] = useState(false)
  const [creating, setCreating] = useState(false)

  // Create form
  const [newBaseId, setNewBaseId] = useState('')
  const [newPreId, setNewPreId] = useState('')
  const [newPostId, setNewPostId] = useState('')
  const [newAxes, setNewAxes] = useState<string[]>(['emphasis', 'scaffolding', 'reading_level', 'misconception'])
  const [newMinFidelity, setNewMinFidelity] = useState(0.85)
  const [newTrigger, setNewTrigger] = useState('pre_quiz')

  // Unit editor
  const [editAxes, setEditAxes] = useState<string[]>([])
  const [editMinFidelity, setEditMinFidelity] = useState(0.85)
  const [editStatus, setEditStatus] = useState('draft')
  const [editTrigger, setEditTrigger] = useState('pre_quiz')
  const [editPreId, setEditPreId] = useState('')
  const [editPostId, setEditPostId] = useState('')
  const [unitEffectiveness, setUnitEffectiveness] = useState<AdaptiveContentEffectiveness | null>(
    null,
  )
  const [effectivenessByUnit, setEffectivenessByUnit] = useState<
    Record<string, AdaptiveContentEffectiveness>
  >({})
  const [keyTerms, setKeyTerms] = useState<AdaptiveContentKeyTerm[]>([])
  const [keyTermInput, setKeyTermInput] = useState('')
  const [savingUnit, setSavingUnit] = useState(false)
  const [refreshingEff, setRefreshingEff] = useState(false)

  // Preview
  const [previewMode, setPreviewMode] = useState<(typeof EMPHASIS_MODES)[number]>('remediate')
  const [preview, setPreview] = useState<AdaptiveContentPreview | null>(null)
  const [previewing, setPreviewing] = useState(false)
  const [editMarkdown, setEditMarkdown] = useState('')
  const [showDiff, setShowDiff] = useState(true)

  // Review queue
  const [queue, setQueue] = useState<AdaptiveContentVariant[]>([])
  const [queueTotal, setQueueTotal] = useState(0)
  const [queueLoading, setQueueLoading] = useState(false)
  const [selectedQueueIds, setSelectedQueueIds] = useState<Set<string>>(new Set())
  const [unitVariants, setUnitVariants] = useState<AdaptiveContentVariant[]>([])

  const contentPages = useMemo(
    () => structure.filter((i) => i.kind === 'content_page'),
    [structure],
  )
  const quizzes = useMemo(() => structure.filter((i) => i.kind === 'quiz'), [structure])
  const titleById = useMemo(() => {
    const m = new Map<string, string>()
    for (const i of structure) m.set(i.id, i.title)
    return m
  }, [structure])

  const selectedUnit = units.find((u) => u.id === selectedUnitId) ?? null
  const budgetExhausted =
    budget != null && !budget.unlimited && (budget.budgetRemaining ?? 0) <= 0

  const load = useCallback(async () => {
    if (!adaptiveContentEnabled) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const [u, s, b, st, eff] = await Promise.all([
        fetchAdaptiveContentUnits(courseCode),
        fetchCourseStructure(courseCode),
        fetchAdaptiveContentBudget(courseCode).catch(() => null),
        fetchAdaptiveContentSettings(courseCode).catch(() => null),
        fetchAdaptiveContentEffectiveness(courseCode).catch(() => [] as AdaptiveContentEffectiveness[]),
      ])
      setUnits(u)
      setStructure(s)
      setBudget(b)
      setSettings(st)
      const byUnit: Record<string, AdaptiveContentEffectiveness> = {}
      for (const e of eff) byUnit[e.unitId] = e
      setEffectivenessByUnit(byUnit)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load adaptive content.')
    } finally {
      setLoading(false)
    }
  }, [adaptiveContentEnabled, courseCode])

  useEffect(() => {
    void load()
  }, [load])

  const loadQueue = useCallback(async () => {
    setQueueLoading(true)
    try {
      const q = await fetchAdaptiveContentReviewQueue(courseCode, { limit: 50, offset: 0 })
      setQueue(q.variants)
      setQueueTotal(q.total)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load review queue.')
    } finally {
      setQueueLoading(false)
    }
  }, [courseCode])

  useEffect(() => {
    if (tab === 'queue' && adaptiveContentEnabled) void loadQueue()
  }, [tab, adaptiveContentEnabled, loadQueue])

  async function openUnit(unit: AdaptiveContentUnit) {
    setSelectedUnitId(unit.id)
    setEditorOpen(true)
    setEditAxes(unit.allowedAxes?.length ? [...unit.allowedAxes] : [])
    setEditMinFidelity(unit.minFidelity ?? 0.85)
    setEditStatus(unit.status)
    setEditTrigger(unit.triggerMode ?? 'pre_quiz')
    setEditPreId(unit.preAssessmentItemId ?? '')
    setEditPostId(unit.postAssessmentItemId ?? '')
    setUnitEffectiveness(effectivenessByUnit[unit.id] ?? null)
    setPreview(null)
    setEditMarkdown('')
    try {
      const [terms, variants, eff] = await Promise.all([
        fetchAdaptiveContentKeyTerms(courseCode, unit.id),
        fetchAdaptiveContentVariants(courseCode, unit.id),
        fetchAdaptiveContentUnitEffectiveness(courseCode, unit.id).catch(() => null),
      ])
      setKeyTerms(terms)
      setUnitVariants(variants)
      if (eff) {
        setUnitEffectiveness(eff)
        setEffectivenessByUnit((prev) => ({ ...prev, [unit.id]: eff }))
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load unit details.')
    }
  }

  async function handleCreate() {
    if (!newBaseId) {
      setError('Pick a content page for the unit.')
      return
    }
    setCreating(true)
    setError(null)
    try {
      const moduleParent = structure.find((i) => i.id === newBaseId)?.parentId
      const unit = await createAdaptiveContentUnit(courseCode, {
        targetKind: 'module',
        targetModuleItemId: moduleParent ?? newBaseId,
        baseContentItemId: newBaseId,
        preAssessmentItemId: newPreId || null,
        postAssessmentItemId: newPostId || null,
        allowedAxes: newAxes,
        status: 'draft',
        triggerMode: newTrigger,
      })
      // min fidelity via patch (create body does not include it)
      await patchAdaptiveContentUnit(courseCode, unit.id, { minFidelity: newMinFidelity })
      setStatusLive('Unit created as draft.')
      setNewBaseId('')
      setNewPreId('')
      setNewPostId('')
      await load()
      await openUnit({ ...unit, minFidelity: newMinFidelity })
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not create unit.')
    } finally {
      setCreating(false)
    }
  }

  async function saveUnitConfig() {
    if (!selectedUnit || !canConfigure) return
    setSavingUnit(true)
    setError(null)
    try {
      if (budgetExhausted && editStatus === 'active' && selectedUnit.status !== 'active') {
        // Allow save but surface warning (AC.5 AC-8).
        setStatusLive(
          'Warning: course adaptive token budget is exhausted — students will see the original until budget resets.',
        )
      }
      const updated = await patchAdaptiveContentUnit(courseCode, selectedUnit.id, {
        allowedAxes: editAxes,
        minFidelity: editMinFidelity,
        status: editStatus,
        triggerMode: editTrigger,
        preAssessmentItemId: editPreId || null,
        clearPreAssessment: !editPreId,
        postAssessmentItemId: editPostId || null,
        clearPostAssessment: !editPostId,
      })
      setUnits((prev) => prev.map((u) => (u.id === updated.id ? { ...u, ...updated } : u)))
      setStatusLive('Unit saved.')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not save unit.')
    } finally {
      setSavingUnit(false)
    }
  }

  async function saveKeyTerms() {
    if (!selectedUnit || !canConfigure) return
    try {
      const terms = await putAdaptiveContentKeyTerms(
        courseCode,
        selectedUnit.id,
        keyTerms.map((t) => ({ term: t.term, mustAppear: t.mustAppear })),
      )
      setKeyTerms(terms)
      setStatusLive('Key terms saved.')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not save key terms.')
    }
  }

  function addKeyTerm() {
    const term = keyTermInput.trim()
    if (!term) return
    if (keyTerms.some((t) => t.term.toLowerCase() === term.toLowerCase())) {
      setKeyTermInput('')
      return
    }
    setKeyTerms((prev) => [
      ...prev,
      {
        id: `tmp-${term}`,
        unitId: selectedUnitId ?? '',
        term,
        mustAppear: true,
      },
    ])
    setKeyTermInput('')
  }

  async function runPreview() {
    if (!selectedUnit) return
    setPreviewing(true)
    setError(null)
    setPreview(null)
    try {
      const result = await previewAdaptiveContentVariant(courseCode, selectedUnit.id, {
        syntheticProfile: {
          emphasisMode: previewMode,
          targetBloom: previewMode === 'compress' ? 'analyze' : 'understand',
          readingLevelPref: 'default',
          modalityPref: 'default',
          axisSet: selectedUnit.allowedAxes,
        },
        persist: true,
      })
      setPreview(result)
      setEditMarkdown(result.variant.variantMarkdown ?? '')
      setStatusLive(
        result.variant.fallback
          ? `Preview fell back to base (${result.variant.fallbackReason || 'generation issue'}).`
          : `Preview ready — fidelity ${fidelityLabel(result.fidelityScore)}.`,
      )
      // refresh unit variants list
      const variants = await fetchAdaptiveContentVariants(courseCode, selectedUnit.id)
      setUnitVariants(variants)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : "Couldn't generate — students see the original.")
    } finally {
      setPreviewing(false)
    }
  }

  async function handleApprove(variant: AdaptiveContentVariant, overrideGate = false) {
    if (!variant.id) return
    try {
      await approveAdaptiveContentVariant(courseCode, variant.id, {
        expectedVariantVersion: variant.variantVersion,
        overrideGate,
      })
      setStatusLive('Variant approved.')
      if (selectedUnit) {
        setUnitVariants(await fetchAdaptiveContentVariants(courseCode, selectedUnit.id))
      }
      if (tab === 'queue') await loadQueue()
      await load()
      setPreview(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Approve failed.')
    }
  }

  async function handleEditApprove(variant: AdaptiveContentVariant) {
    if (!variant.id) return
    try {
      await editApproveAdaptiveContentVariant(courseCode, variant.id, {
        variantMarkdown: editMarkdown,
        expectedVariantVersion: variant.variantVersion,
      })
      setStatusLive('Variant edited and approved.')
      if (selectedUnit) {
        setUnitVariants(await fetchAdaptiveContentVariants(courseCode, selectedUnit.id))
      }
      await load()
      setPreview(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Edit-and-approve failed.')
    }
  }

  async function handleReject(variant: AdaptiveContentVariant, note?: string) {
    if (!variant.id) return
    try {
      await rejectAdaptiveContentVariant(courseCode, variant.id, {
        expectedVariantVersion: variant.variantVersion,
        note,
      })
      setStatusLive('Variant rejected.')
      if (selectedUnit) {
        setUnitVariants(await fetchAdaptiveContentVariants(courseCode, selectedUnit.id))
      }
      if (tab === 'queue') await loadQueue()
      await load()
      setPreview(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Reject failed.')
    }
  }

  async function handleRevoke(variant: AdaptiveContentVariant) {
    if (!variant.id || !canConfigure) return
    try {
      await revokeAdaptiveContentVariant(courseCode, variant.id, {
        expectedVariantVersion: variant.variantVersion,
      })
      setStatusLive('Variant revoked; students revert to base.')
      if (selectedUnit) {
        setUnitVariants(await fetchAdaptiveContentVariants(courseCode, selectedUnit.id))
      }
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Revoke failed.')
    }
  }

  async function handleBulk(action: 'approve' | 'reject') {
    const ids = [...selectedQueueIds]
    if (ids.length === 0) return
    // Group by unit for the bulk endpoint (per-unit).
    const byUnit = new Map<string, string[]>()
    for (const v of queue) {
      if (v.id && selectedQueueIds.has(v.id) && v.unitId) {
        const list = byUnit.get(v.unitId) ?? []
        list.push(v.id)
        byUnit.set(v.unitId, list)
      }
    }
    const ok = await confirm({
      title: `${action === 'approve' ? 'Approve' : 'Reject'} ${ids.length} variant(s)?`,
      description: 'Check fidelity badges first before bulk review.',
      confirmLabel: action === 'approve' ? 'Approve' : 'Reject',
      variant: action === 'reject' ? 'danger' : 'default',
    })
    if (!ok) return
    try {
      for (const [unitId, variantIds] of byUnit) {
        await bulkReviewAdaptiveContentVariants(courseCode, unitId, {
          action,
          variantIds,
        })
      }
      setSelectedQueueIds(new Set())
      setStatusLive(`Bulk ${action} finished.`)
      await loadQueue()
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Bulk action failed.')
    }
  }

  async function handlePrewarm(unitId: string) {
    if (!canConfigure) return
    try {
      const r = await prewarmAdaptiveContentUnit(courseCode, unitId)
      setStatusLive(`Pre-warm enqueued ${r.enqueued} job(s).`)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Pre-warm failed.')
    }
  }

  async function toggleRequireApproval(next: boolean) {
    if (!settings || !canConfigure) return
    try {
      const updated = await putAdaptiveContentSettings(courseCode, {
        ...settings,
        requireInstructorApproval: next,
      })
      setSettings(updated)
      setStatusLive(
        next
          ? 'Instructor approval required before variants serve.'
          : 'Gate-passing variants may auto-serve.',
      )
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not update approval setting.')
    }
  }

  if (!adaptiveContentEnabled) {
    return (
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
        <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-50">
          Adaptive Content
        </h2>
        <p className="mt-2 text-sm text-slate-600 dark:text-neutral-300">
          Turn on Adaptive Content and add your first unit.
        </p>
        <p className="mt-3 text-sm">
          <Link
            to={`/courses/${encodeURIComponent(courseCode)}/settings/features`}
            className="font-medium text-indigo-600 hover:text-indigo-500"
          >
            Enable under Features → Adaptive Content
          </Link>
        </p>
      </section>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-50">
            Adaptive Content workspace
          </h2>
          <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
            Configure units, preview learner archetypes, and review generated variants.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {budget && (
            <span
              className={`rounded-full px-2.5 py-1 text-xs font-medium ${
                budgetExhausted
                  ? 'bg-amber-100 text-amber-900 dark:bg-amber-950/40 dark:text-amber-200'
                  : 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-200'
              }`}
              title={`Period starting ${budget.periodStart}`}
            >
              Budget:{' '}
              {budget.unlimited
                ? 'unlimited'
                : `${budget.tokensUsedPeriod.toLocaleString()} / ${budget.monthlyTokenBudget.toLocaleString()} tokens`}
              {budget.generationPaused ? ' · paused' : ''}
            </span>
          )}
          <button
            type="button"
            onClick={() => void load()}
            className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
          >
            <RefreshCw className="h-3.5 w-3.5" aria-hidden />
            Refresh
          </button>
        </div>
      </div>

      {budgetExhausted && (
        <div
          className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-100"
          role="status"
        >
          Course adaptive token budget is exhausted. Activating units will not generate new variants
          until the budget resets — students will see the original content.
        </div>
      )}

      {settings && canConfigure && (
        <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
          <input
            type="checkbox"
            checked={settings.requireInstructorApproval}
            onChange={(e) => void toggleRequireApproval(e.target.checked)}
          />
          Require instructor approval before variants serve (otherwise gate-passing variants
          auto-serve)
        </label>
      )}

      <div className="flex gap-2 border-b border-slate-200 dark:border-neutral-800" role="tablist">
        <button
          type="button"
          role="tab"
          aria-selected={tab === 'units'}
          onClick={() => setTab('units')}
          className={`px-3 py-2 text-sm font-medium ${
            tab === 'units'
              ? 'border-b-2 border-indigo-600 text-indigo-700 dark:text-indigo-300'
              : 'text-slate-600 dark:text-neutral-400'
          }`}
        >
          Units
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={tab === 'queue'}
          onClick={() => setTab('queue')}
          className={`px-3 py-2 text-sm font-medium ${
            tab === 'queue'
              ? 'border-b-2 border-indigo-600 text-indigo-700 dark:text-indigo-300'
              : 'text-slate-600 dark:text-neutral-400'
          }`}
        >
          Review queue
          {queueTotal > 0 ? ` (${queueTotal})` : ''}
        </button>
      </div>

      {error && (
        <p className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/30 dark:text-rose-100" role="alert">
          {error}
        </p>
      )}
      <div className="sr-only" role="status" aria-live="polite">
        {statusLive}
      </div>
      {statusLive && (
        <p className="text-sm text-emerald-700 dark:text-emerald-400" role="status">
          {statusLive}
        </p>
      )}

      {loading ? (
        <div className="space-y-2" aria-busy="true">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-12 animate-pulse rounded-xl bg-slate-100 dark:bg-neutral-800"
            />
          ))}
        </div>
      ) : tab === 'units' ? (
        <>
          {canConfigure && (
            <section className="rounded-2xl border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-50">
                Create unit
              </h3>
              <div className="mt-3 grid gap-3 sm:grid-cols-2">
                <label className="flex flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
                  Content page
                  <select
                    className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                    value={newBaseId}
                    onChange={(e) => setNewBaseId(e.target.value)}
                  >
                    <option value="">Select…</option>
                    {contentPages.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.title}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="flex flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
                  Pre-check quiz (optional)
                  <select
                    className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                    value={newPreId}
                    onChange={(e) => setNewPreId(e.target.value)}
                  >
                    <option value="">None</option>
                    {quizzes.map((q) => (
                      <option key={q.id} value={q.id}>
                        {q.title}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="flex flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
                  Exit ticket (post-assessment)
                  <select
                    className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                    value={newPostId}
                    onChange={(e) => setNewPostId(e.target.value)}
                    data-testid="ace-new-post-assessment"
                  >
                    <option value="">None</option>
                    {quizzes.map((q) => (
                      <option key={q.id} value={q.id}>
                        {q.title}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="flex flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
                  Min fidelity
                  <div className="flex items-center gap-2">
                    <input
                      type="range"
                      min={0}
                      max={1}
                      step={0.05}
                      value={newMinFidelity}
                      onChange={(e) => setNewMinFidelity(Number(e.target.value))}
                      className="flex-1"
                      aria-valuetext={`${Math.round(newMinFidelity * 100)} percent`}
                    />
                    <input
                      type="number"
                      min={0}
                      max={1}
                      step={0.05}
                      value={newMinFidelity}
                      onChange={(e) => setNewMinFidelity(Number(e.target.value))}
                      className="w-16 rounded border border-slate-200 px-1 py-0.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                      aria-label="Min fidelity numeric"
                    />
                  </div>
                </label>
                <label className="flex flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
                  Trigger mode
                  <select
                    className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                    value={newTrigger}
                    onChange={(e) => setNewTrigger(e.target.value)}
                  >
                    <option value="pre_quiz">Pre-quiz</option>
                    <option value="diagnostic_first_visit">Diagnostic first visit</option>
                    <option value="mastery_snapshot">Mastery snapshot</option>
                  </select>
                </label>
                <fieldset className="sm:col-span-2">
                  <legend className="text-xs font-medium text-slate-700 dark:text-neutral-300">
                    Allowed axes
                  </legend>
                  <div className="mt-1 flex flex-wrap gap-3">
                    {ALL_AXES.map((ax) => (
                      <label key={ax.id} className="flex items-center gap-1.5 text-sm text-slate-700 dark:text-neutral-300">
                        <input
                          type="checkbox"
                          checked={newAxes.includes(ax.id)}
                          onChange={(e) => {
                            setNewAxes((prev) =>
                              e.target.checked
                                ? [...prev, ax.id]
                                : prev.filter((x) => x !== ax.id),
                            )
                          }}
                        />
                        {ax.label}
                      </label>
                    ))}
                  </div>
                </fieldset>
              </div>
              <button
                type="button"
                disabled={creating || !newBaseId}
                onClick={() => void handleCreate()}
                className="mt-3 inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
              >
                {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                Create unit
              </button>
            </section>
          )}

          {units.length === 0 ? (
            <p className="py-8 text-center text-sm text-slate-500 dark:text-neutral-400">
              Turn on Adaptive Content and add your first unit.
            </p>
          ) : (
            <>
              <div className="mb-2 flex justify-end">
                {canConfigure && (
                  <button
                    type="button"
                    disabled={refreshingEff}
                    data-testid="ace-refresh-effectiveness"
                    onClick={() => {
                      void (async () => {
                        setRefreshingEff(true)
                        try {
                          await refreshAdaptiveContentEffectiveness(courseCode)
                          const eff = await fetchAdaptiveContentEffectiveness(courseCode)
                          const byUnit: Record<string, AdaptiveContentEffectiveness> = {}
                          for (const e of eff) byUnit[e.unitId] = e
                          setEffectivenessByUnit(byUnit)
                          setStatusLive('Effectiveness refreshed.')
                        } catch (e) {
                          setError(
                            e instanceof Error ? e.message : 'Could not refresh effectiveness.',
                          )
                        } finally {
                          setRefreshingEff(false)
                        }
                      })()
                    }}
                    className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-2 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-900"
                  >
                    {refreshingEff ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <RefreshCw className="h-3.5 w-3.5" />
                    )}
                    Refresh effectiveness
                  </button>
                )}
              </div>
              <div className="overflow-x-auto rounded-2xl border border-slate-200 dark:border-neutral-800">
                <table className="min-w-full text-left text-sm">
                  <thead className="bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:bg-neutral-900 dark:text-neutral-400">
                    <tr>
                      <th className="px-3 py-2">Content page</th>
                      <th className="px-3 py-2">Status</th>
                      <th className="px-3 py-2">Effectiveness</th>
                      <th className="px-3 py-2">Coverage</th>
                      <th className="px-3 py-2">Axes</th>
                      <th className="px-3 py-2">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                    {units.map((u) => (
                      <tr key={u.id} className="bg-white dark:bg-neutral-950">
                        <td className="px-3 py-2 font-medium text-slate-900 dark:text-neutral-100">
                          {titleById.get(u.baseContentItemId) ?? u.baseContentItemId.slice(0, 8)}
                        </td>
                        <td className="px-3 py-2">
                          <span
                            className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusBadgeClass(u.status)}`}
                          >
                            {u.status}
                          </span>
                        </td>
                        <td className="px-3 py-2">
                          <EffectivenessChip
                            effectiveness={
                              u.postAssessmentItemId ? effectivenessByUnit[u.id] ?? null : null
                            }
                            compact
                          />
                        </td>
                        <td className="px-3 py-2 text-slate-600 dark:text-neutral-300">
                          {u.variantTotal ?? 0} total · {u.variantApproved ?? 0} approved ·{' '}
                          {u.variantPendingReview ?? 0} pending
                        </td>
                        <td className="px-3 py-2 text-xs text-slate-500 dark:text-neutral-400">
                          {(u.allowedAxes ?? []).join(', ') || '—'}
                        </td>
                        <td className="px-3 py-2">
                          <div className="flex flex-wrap gap-2">
                            <button
                              type="button"
                              onClick={() => void openUnit(u)}
                              className="text-xs font-medium text-indigo-600 hover:text-indigo-500"
                            >
                              Open
                            </button>
                            {canConfigure && u.status === 'active' && (
                              <button
                                type="button"
                                onClick={() => void handlePrewarm(u.id)}
                                className="text-xs font-medium text-slate-600 hover:text-slate-800 dark:text-neutral-400"
                              >
                                Pre-warm now
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </>
      ) : (
        <section className="rounded-2xl border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-50">
              Pending review ({queueTotal})
            </h3>
            <div className="flex gap-2">
              <button
                type="button"
                disabled={selectedQueueIds.size === 0}
                onClick={() => void handleBulk('approve')}
                className="rounded-lg bg-emerald-600 px-2.5 py-1 text-xs font-medium text-white disabled:opacity-50"
              >
                Bulk approve
              </button>
              <button
                type="button"
                disabled={selectedQueueIds.size === 0}
                onClick={() => void handleBulk('reject')}
                className="rounded-lg bg-rose-600 px-2.5 py-1 text-xs font-medium text-white disabled:opacity-50"
              >
                Bulk reject
              </button>
            </div>
          </div>
          {queueLoading ? (
            <p className="mt-4 text-sm text-slate-500">Loading queue…</p>
          ) : queue.length === 0 ? (
            <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">
              No variants awaiting review.
            </p>
          ) : (
            <ul className="mt-3 divide-y divide-slate-100 dark:divide-neutral-800">
              {queue.map((v) => (
                <li key={v.id} className="flex flex-wrap items-start gap-3 py-3">
                  <input
                    type="checkbox"
                    checked={!!v.id && selectedQueueIds.has(v.id)}
                    onChange={(e) => {
                      if (!v.id) return
                      setSelectedQueueIds((prev) => {
                        const next = new Set(prev)
                        if (e.target.checked) next.add(v.id!)
                        else next.delete(v.id!)
                        return next
                      })
                    }}
                    aria-label={`Select variant ${v.profileSignature}`}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-xs text-slate-600 dark:text-neutral-400">
                        {v.profileSignature.slice(0, 16)}…
                      </span>
                      <span className={`rounded-full px-2 py-0.5 text-xs ${statusBadgeClass(v.status)}`}>
                        {v.status}
                      </span>
                      <span className="text-xs text-slate-500">
                        fidelity {fidelityLabel(v.fidelityScore)}
                      </span>
                      {(v.safetyFlags?.length ?? 0) > 0 && (
                        <span className="text-xs text-rose-600">safety: {v.safetyFlags?.join(', ')}</span>
                      )}
                      {(v.a11yFlags?.length ?? 0) > 0 && (
                        <span className="text-xs text-amber-700">a11y: {v.a11yFlags?.join(', ')}</span>
                      )}
                    </div>
                    <p className="mt-1 line-clamp-2 text-xs text-slate-600 dark:text-neutral-400">
                      {(v.variantMarkdown ?? '').slice(0, 160)}
                    </p>
                  </div>
                  <div className="flex gap-1">
                    <button
                      type="button"
                      className="rounded p-1 text-emerald-700 hover:bg-emerald-50 dark:text-emerald-300"
                      aria-label="Approve"
                      onClick={() => void handleApprove(v)}
                    >
                      <Check className="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      className="rounded p-1 text-rose-700 hover:bg-rose-50 dark:text-rose-300"
                      aria-label="Reject"
                      onClick={() => void handleReject(v, 'Rejected from queue')}
                    >
                      <X className="h-4 w-4" />
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
      )}

      {/* Unit editor drawer */}
      {editorOpen && selectedUnit && (
        <div
          className="fixed inset-0 z-40 flex justify-end bg-black/40"
          role="dialog"
          aria-modal="true"
          aria-labelledby="ace-unit-editor-title"
        >
          <button
            type="button"
            className="flex-1 cursor-default"
            aria-label="Close unit editor"
            onClick={() => setEditorOpen(false)}
          />
          <div className="flex h-full w-full max-w-3xl flex-col overflow-y-auto bg-white shadow-xl dark:bg-neutral-900">
            <div className="sticky top-0 z-10 flex items-center justify-between border-b border-slate-200 bg-white px-4 py-3 dark:border-neutral-800 dark:bg-neutral-900">
              <h3 id="ace-unit-editor-title" className="text-sm font-semibold text-slate-900 dark:text-neutral-50">
                {titleById.get(selectedUnit.baseContentItemId) ?? 'Unit'}
              </h3>
              <button
                type="button"
                onClick={() => setEditorOpen(false)}
                className="rounded p-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
                aria-label="Close"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="space-y-6 p-4">
              {unitEffectiveness && (
                <section
                  className="rounded-lg border border-slate-200 bg-slate-50 p-3 dark:border-neutral-800 dark:bg-neutral-900"
                  data-testid="ace-verdict-banner"
                  aria-live="polite"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                      Effectiveness
                    </span>
                    <EffectivenessChip effectiveness={unitEffectiveness} />
                  </div>
                  <EffectivenessSummaryTable effectiveness={unitEffectiveness} />
                  {unitEffectiveness.byMode.length > 0 && (
                    <div className="mt-2">
                      <p className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                        By emphasis mode
                      </p>
                      <ul className="mt-1 space-y-0.5 text-xs text-slate-600 dark:text-neutral-400">
                        {unitEffectiveness.byMode.map((m) => (
                          <li key={m.emphasisMode}>
                            {m.emphasisMode}: n={m.n}
                            {m.meanLift != null
                              ? `, mean lift ${Math.round(m.meanLift)} pts`
                              : ' (suppressed)'}
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}
                </section>
              )}
              {canConfigure ? (
                <section className="space-y-3">
                  <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Configuration
                  </h4>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <label className="flex flex-col gap-1 text-xs font-medium">
                      Status
                      <select
                        value={editStatus}
                        onChange={(e) => setEditStatus(e.target.value)}
                        className="rounded-lg border border-slate-200 px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                      >
                        <option value="draft">draft</option>
                        <option value="active">active</option>
                        <option value="paused">paused</option>
                        <option value="archived">archived</option>
                      </select>
                    </label>
                    <label className="flex flex-col gap-1 text-xs font-medium">
                      Trigger
                      <select
                        value={editTrigger}
                        onChange={(e) => setEditTrigger(e.target.value)}
                        className="rounded-lg border border-slate-200 px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                      >
                        <option value="pre_quiz">pre_quiz</option>
                        <option value="diagnostic_first_visit">diagnostic_first_visit</option>
                        <option value="mastery_snapshot">mastery_snapshot</option>
                      </select>
                    </label>
                    <label className="flex flex-col gap-1 text-xs font-medium sm:col-span-2">
                      Pre-check quiz
                      <select
                        value={editPreId}
                        onChange={(e) => setEditPreId(e.target.value)}
                        className="rounded-lg border border-slate-200 px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                      >
                        <option value="">None</option>
                        {quizzes.map((q) => (
                          <option key={q.id} value={q.id}>
                            {q.title}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="flex flex-col gap-1 text-xs font-medium sm:col-span-2">
                      Exit ticket (post-assessment)
                      <select
                        value={editPostId}
                        onChange={(e) => setEditPostId(e.target.value)}
                        className="rounded-lg border border-slate-200 px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                        data-testid="ace-edit-post-assessment"
                      >
                        <option value="">None</option>
                        {quizzes.map((q) => (
                          <option key={q.id} value={q.id}>
                            {q.title}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="flex flex-col gap-1 text-xs font-medium sm:col-span-2">
                      Min fidelity ({Math.round(editMinFidelity * 100)}%)
                      <div className="flex items-center gap-2">
                        <input
                          type="range"
                          min={0}
                          max={1}
                          step={0.05}
                          value={editMinFidelity}
                          onChange={(e) => setEditMinFidelity(Number(e.target.value))}
                          className="flex-1"
                        />
                        <input
                          type="number"
                          min={0}
                          max={1}
                          step={0.05}
                          value={editMinFidelity}
                          onChange={(e) => setEditMinFidelity(Number(e.target.value))}
                          className="w-16 rounded border border-slate-200 px-1 py-0.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                        />
                      </div>
                    </label>
                  </div>
                  <fieldset>
                    <legend className="text-xs font-medium">Allowed axes</legend>
                    <div className="mt-1 flex flex-wrap gap-3">
                      {ALL_AXES.map((ax) => (
                        <label key={ax.id} className="flex items-center gap-1.5 text-sm">
                          <input
                            type="checkbox"
                            checked={editAxes.includes(ax.id)}
                            onChange={(e) => {
                              setEditAxes((prev) =>
                                e.target.checked
                                  ? [...prev, ax.id]
                                  : prev.filter((x) => x !== ax.id),
                              )
                            }}
                          />
                          {ax.label}
                        </label>
                      ))}
                    </div>
                  </fieldset>
                  <div>
                    <label className="text-xs font-medium">Key terms (must appear)</label>
                    <div className="mt-1 flex flex-wrap gap-1">
                      {keyTerms.map((t) => (
                        <button
                          key={t.id}
                          type="button"
                          onClick={() =>
                            setKeyTerms((prev) => prev.filter((x) => x.term !== t.term))
                          }
                          className="rounded-full bg-slate-100 px-2 py-0.5 text-xs dark:bg-neutral-800"
                          title="Remove"
                        >
                          {t.term} ×
                        </button>
                      ))}
                    </div>
                    <div className="mt-2 flex gap-2">
                      <input
                        type="text"
                        value={keyTermInput}
                        onChange={(e) => setKeyTermInput(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            e.preventDefault()
                            addKeyTerm()
                          }
                        }}
                        placeholder="Add term…"
                        className="flex-1 rounded-lg border border-slate-200 px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                      />
                      <button
                        type="button"
                        onClick={addKeyTerm}
                        className="rounded-lg border border-slate-200 px-2 py-1 text-xs dark:border-neutral-700"
                      >
                        Add
                      </button>
                      <button
                        type="button"
                        onClick={() => void saveKeyTerms()}
                        className="rounded-lg bg-slate-800 px-2 py-1 text-xs text-white dark:bg-neutral-200 dark:text-neutral-900"
                      >
                        Save terms
                      </button>
                    </div>
                  </div>
                  <button
                    type="button"
                    disabled={savingUnit}
                    onClick={() => void saveUnitConfig()}
                    className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
                  >
                    {savingUnit ? 'Saving…' : 'Save configuration'}
                  </button>
                </section>
              ) : (
                <p className="text-sm text-slate-500">
                  Review-only access: you can approve or reject variants but cannot change unit
                  settings or budgets.
                </p>
              )}

              <section className="space-y-3">
                <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                  Preview
                </h4>
                <div className="flex flex-wrap items-end gap-2">
                  <label className="flex flex-col gap-1 text-xs font-medium">
                    Preview as…
                    <select
                      value={previewMode}
                      onChange={(e) =>
                        setPreviewMode(e.target.value as (typeof EMPHASIS_MODES)[number])
                      }
                      className="rounded-lg border border-slate-200 px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                    >
                      {EMPHASIS_MODES.map((m) => (
                        <option key={m} value={m}>
                          {m} learner
                        </option>
                      ))}
                    </select>
                  </label>
                  <button
                    type="button"
                    disabled={previewing}
                    onClick={() => void runPreview()}
                    className="inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
                  >
                    {previewing ? (
                      <>
                        <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                        Checking fidelity…
                      </>
                    ) : (
                      <>
                        <Sparkles className="h-4 w-4" aria-hidden />
                        Generate preview
                      </>
                    )}
                  </button>
                  <label className="flex items-center gap-1.5 text-xs text-slate-600 dark:text-neutral-400">
                    <input
                      type="checkbox"
                      checked={showDiff}
                      onChange={(e) => setShowDiff(e.target.checked)}
                    />
                    Show diff
                  </label>
                </div>

                {preview && (
                  <div className="space-y-3">
                    <div className="flex flex-wrap gap-2 text-xs">
                      <span className={`rounded-full px-2 py-0.5 font-medium ${statusBadgeClass(preview.variant.status)}`}>
                        {preview.variant.status}
                      </span>
                      <span className="rounded-full bg-slate-100 px-2 py-0.5 dark:bg-neutral-800">
                        fidelity {fidelityLabel(preview.fidelityScore)}
                      </span>
                      <span className="rounded-full bg-slate-100 px-2 py-0.5 dark:bg-neutral-800">
                        tokens {(preview.promptTokens ?? 0) + (preview.completionTokens ?? 0)}
                      </span>
                      {(preview.a11yFlags?.length ?? 0) > 0 && (
                        <span className="rounded-full bg-amber-100 px-2 py-0.5 text-amber-900">
                          a11y: {preview.a11yFlags.join(', ')}
                        </span>
                      )}
                      {(preview.safetyFlags?.length ?? 0) > 0 && (
                        <span className="rounded-full bg-rose-100 px-2 py-0.5 text-rose-900">
                          safety: {preview.safetyFlags.join(', ')}
                        </span>
                      )}
                    </div>
                    {showDiff && preview.baseMarkdown != null ? (
                      <VariantDiff
                        baseMarkdown={preview.baseMarkdown}
                        variantMarkdown={preview.variant.variantMarkdown ?? ''}
                        className="max-h-80"
                      />
                    ) : (
                      <div className="grid gap-3 md:grid-cols-2">
                        <div>
                          <p className="mb-1 text-xs font-medium text-slate-500">Base</p>
                          <pre className="max-h-64 overflow-auto whitespace-pre-wrap rounded-lg border border-slate-200 p-2 text-xs dark:border-neutral-700">
                            {preview.baseMarkdown ?? '—'}
                          </pre>
                        </div>
                        <div>
                          <p className="mb-1 text-xs font-medium text-slate-500">Variant</p>
                          <pre className="max-h-64 overflow-auto whitespace-pre-wrap rounded-lg border border-slate-200 p-2 text-xs dark:border-neutral-700">
                            {preview.variant.variantMarkdown ?? '—'}
                          </pre>
                        </div>
                      </div>
                    )}
                    {preview.variant.id && preview.variant.status === 'pending_review' && (
                      <div className="space-y-2">
                        <label className="block text-xs font-medium">
                          Edit before approve
                          <textarea
                            value={editMarkdown}
                            onChange={(e) => setEditMarkdown(e.target.value)}
                            rows={6}
                            className="mt-1 w-full rounded-lg border border-slate-200 p-2 font-mono text-xs dark:border-neutral-700 dark:bg-neutral-950"
                          />
                        </label>
                        <div className="flex flex-wrap gap-2">
                          <button
                            type="button"
                            onClick={() => void handleApprove(preview.variant)}
                            className="rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white"
                          >
                            Approve
                          </button>
                          <button
                            type="button"
                            onClick={() => void handleEditApprove(preview.variant)}
                            className="rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white"
                          >
                            Edit &amp; approve
                          </button>
                          <button
                            type="button"
                            onClick={() => void handleReject(preview.variant)}
                            className="rounded-lg bg-rose-600 px-3 py-1.5 text-xs font-medium text-white"
                          >
                            Reject
                          </button>
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </section>

              <section>
                <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                  Variants ({unitVariants.length})
                </h4>
                <ul className="mt-2 divide-y divide-slate-100 dark:divide-neutral-800">
                  {unitVariants.map((v) => (
                    <li key={v.id ?? v.profileSignature} className="flex flex-wrap items-center justify-between gap-2 py-2 text-xs">
                      <div>
                        <span className="font-mono">{v.profileSignature.slice(0, 12)}…</span>{' '}
                        <span className={`rounded-full px-1.5 py-0.5 ${statusBadgeClass(v.status)}`}>
                          {v.status}
                        </span>
                        {v.humanEdited ? ' · human-edited' : ''}
                        {' · '}
                        fidelity {fidelityLabel(v.fidelityScore)}
                      </div>
                      <div className="flex gap-2">
                        {v.status === 'pending_review' && v.id && (
                          <>
                            <button
                              type="button"
                              className="text-emerald-700 dark:text-emerald-300"
                              onClick={() => void handleApprove(v)}
                            >
                              Approve
                            </button>
                            <button
                              type="button"
                              className="text-rose-700 dark:text-rose-300"
                              onClick={() => void handleReject(v)}
                            >
                              Reject
                            </button>
                          </>
                        )}
                        {(v.status === 'approved' || v.status === 'auto_served') &&
                          canConfigure &&
                          v.id && (
                            <button
                              type="button"
                              className="text-amber-700 dark:text-amber-300"
                              onClick={() => void handleRevoke(v)}
                            >
                              Revoke
                            </button>
                          )}
                      </div>
                    </li>
                  ))}
                  {unitVariants.length === 0 && (
                    <li className="py-2 text-slate-500">No variants yet — run a preview.</li>
                  )}
                </ul>
              </section>
            </div>
          </div>
        </div>
      )}
      {ConfirmDialogHost}
    </div>
  )
}
