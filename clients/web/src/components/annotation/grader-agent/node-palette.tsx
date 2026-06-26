import { useMemo, useRef, useState, type DragEvent, type KeyboardEvent, type ReactNode } from 'react'
import {
  BookOpen,
  ClipboardList,
  Code2,
  FileText,
  Flag,
  GitBranch,
  GraduationCap,
  GripVertical,
  Hash,
  ListChecks,
  Lock,
  Search,
  ShieldCheck,
  Sigma,
  Sparkles,
  Target,
  UserCheck,
  ListOrdered,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { beginPaletteDrag } from './palette-drag'
import type { GradingAgentItemKind, PaletteNodeType } from './types'

export const GRADER_AGENT_DRAG_MIME = 'text/plain'

type NodePaletteProps = {
  disabled?: boolean
  codeExecutionEnabled?: boolean
  itemKind?: GradingAgentItemKind
  onAddNode: (type: PaletteNodeType) => void
}

type PaletteItemConfig = {
  type: PaletteNodeType
  labelKey: string
  descriptionKey: string
  icon: LucideIcon
  iconClass: string
}

const PALETTE_SURFACE_CLASS =
  'rounded-xl bg-white shadow-sm ring-1 ring-black/[0.05] hover:shadow-[0_0_0_1px_rgba(0,0,0,0.08),0_1px_2px_-1px_rgba(0,0,0,0.08),0_2px_4px_0_rgba(0,0,0,0.04)] motion-safe:transition-[box-shadow,transform] motion-safe:active:scale-[0.98] dark:bg-neutral-900 dark:ring-white/10 dark:hover:ring-white/[0.13]'

const INPUT_ITEMS: PaletteItemConfig[] = [
  {
    type: 'studentSubmission',
    labelKey: 'gradingAgent.canvas.palette.studentSubmission',
    descriptionKey: 'gradingAgent.canvas.palette.description.studentSubmission',
    icon: FileText,
    iconClass: 'bg-slate-500/10 text-slate-600 dark:bg-neutral-500/15 dark:text-neutral-300',
  },
  {
    type: 'activity',
    labelKey: 'gradingAgent.canvas.palette.activity',
    descriptionKey: 'gradingAgent.canvas.palette.description.activity',
    icon: ClipboardList,
    iconClass: 'bg-amber-500/10 text-amber-700 dark:text-amber-300',
  },
  {
    type: 'reference',
    labelKey: 'gradingAgent.canvas.palette.reference',
    descriptionKey: 'gradingAgent.canvas.palette.description.reference',
    icon: BookOpen,
    iconClass: 'bg-violet-500/10 text-violet-700 dark:text-violet-300',
  },
  {
    type: 'rubric',
    labelKey: 'gradingAgent.canvas.palette.rubric',
    descriptionKey: 'gradingAgent.canvas.palette.description.rubric',
    icon: ListChecks,
    iconClass: 'bg-orange-500/10 text-orange-700 dark:text-orange-300',
  },
]

const PROCESSING_ITEMS: PaletteItemConfig[] = [
  {
    type: 'ai',
    labelKey: 'gradingAgent.canvas.palette.ai',
    descriptionKey: 'gradingAgent.canvas.palette.description.ai',
    icon: Sparkles,
    iconClass: 'bg-indigo-500/10 text-indigo-700 dark:text-indigo-300',
  },
  {
    type: 'criterionGrader',
    labelKey: 'gradingAgent.canvas.palette.criterionGrader',
    descriptionKey: 'gradingAgent.canvas.palette.description.criterionGrader',
    icon: Target,
    iconClass: 'bg-indigo-500/10 text-indigo-700 dark:text-indigo-300',
  },
  {
    type: 'codeTestRunner',
    labelKey: 'gradingAgent.canvas.palette.codeTests',
    descriptionKey: 'gradingAgent.canvas.palette.description.codeTests',
    icon: Code2,
    iconClass: 'bg-cyan-500/10 text-cyan-700 dark:text-cyan-300',
  },
  {
    type: 'conditionalRouter',
    labelKey: 'gradingAgent.canvas.palette.router',
    descriptionKey: 'gradingAgent.canvas.palette.description.router',
    icon: GitBranch,
    iconClass: 'bg-slate-500/10 text-slate-600 dark:text-neutral-300',
  },
  {
    type: 'scoreAggregator',
    labelKey: 'gradingAgent.canvas.palette.aggregator',
    descriptionKey: 'gradingAgent.canvas.palette.description.aggregator',
    icon: Sigma,
    iconClass: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
  },
  {
    type: 'humanReviewGate',
    labelKey: 'gradingAgent.canvas.palette.reviewGate',
    descriptionKey: 'gradingAgent.canvas.palette.description.reviewGate',
    icon: UserCheck,
    iconClass: 'bg-slate-500/10 text-slate-600 dark:text-neutral-300',
  },
  {
    type: 'originality',
    labelKey: 'gradingAgent.canvas.palette.originality',
    descriptionKey: 'gradingAgent.canvas.palette.description.originality',
    icon: ShieldCheck,
    iconClass: 'bg-amber-500/10 text-amber-700 dark:text-amber-300',
  },
  {
    type: 'setScore',
    labelKey: 'gradingAgent.canvas.palette.setScore',
    descriptionKey: 'gradingAgent.canvas.palette.description.setScore',
    icon: Hash,
    iconClass: 'bg-teal-500/10 text-teal-700 dark:text-teal-300',
  },
]

const OUTPUT_ITEMS: PaletteItemConfig[] = [
  {
    type: 'flagForReview',
    labelKey: 'gradingAgent.canvas.palette.flagForReview',
    descriptionKey: 'gradingAgent.canvas.palette.description.flagForReview',
    icon: Flag,
    iconClass: 'bg-rose-500/10 text-rose-700 dark:text-rose-300',
  },
]

function normalizeSearch(value: string): string {
  return value.trim().toLowerCase()
}

function PaletteGroup({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-1.5">
      <h3 className="px-0.5 text-[11px] font-semibold uppercase tracking-wide text-slate-400 dark:text-neutral-500">
        {title}
      </h3>
      <div className="flex flex-col gap-1.5">{children}</div>
    </section>
  )
}

function PaletteItem({
  type,
  label,
  description,
  icon: Icon,
  iconClass,
  disabled,
  onAddNode,
}: {
  type: PaletteNodeType
  label: string
  description: string
  icon: LucideIcon
  iconClass: string
  disabled?: boolean
  onAddNode: (type: PaletteNodeType) => void
}) {
  const draggedRef = useRef(false)

  const onDragStart = (event: DragEvent<HTMLDivElement>) => {
    if (disabled) {
      event.preventDefault()
      return
    }
    draggedRef.current = true
    beginPaletteDrag(type)
    event.dataTransfer.setData(GRADER_AGENT_DRAG_MIME, type)
    event.dataTransfer.effectAllowed = 'move'
  }

  const onDragEnd = () => {
    window.setTimeout(() => {
      draggedRef.current = false
    }, 100)
  }

  return (
    <div
      role="button"
      tabIndex={disabled ? -1 : 0}
      draggable={!disabled}
      aria-disabled={disabled}
      aria-label={label}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      onClick={() => {
        if (disabled || draggedRef.current) return
        onAddNode(type)
      }}
      onKeyDown={(event: KeyboardEvent<HTMLDivElement>) => {
        if (disabled) return
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          onAddNode(type)
        }
      }}
      className={`group relative flex min-h-10 cursor-grab items-start gap-2.5 px-2.5 py-2.5 active:cursor-grabbing aria-disabled:cursor-not-allowed aria-disabled:opacity-50 ${PALETTE_SURFACE_CLASS}`}
    >
      <span
        className={`flex size-8 shrink-0 items-center justify-center rounded-lg ${iconClass}`}
        aria-hidden
      >
        <Icon className="size-4" strokeWidth={2} />
      </span>
      <span className="min-w-0 flex-1 pt-0.5">
        <span className="text-pretty text-sm font-medium leading-snug text-slate-800 dark:text-neutral-100">
          {label}
        </span>
        <span className="mt-0.5 block text-pretty text-xs leading-snug text-slate-500 dark:text-neutral-400">
          {description}
        </span>
      </span>
      <GripVertical
        className="mt-1 size-4 shrink-0 text-slate-300 opacity-0 motion-safe:transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100 dark:text-neutral-600"
        aria-hidden
      />
    </div>
  )
}

function PaletteFixedItem({
  label,
  description,
  badge,
  icon: Icon,
  iconClass,
}: {
  label: string
  description: string
  badge: string
  icon: LucideIcon
  iconClass: string
}) {
  return (
    <div
      className="flex min-h-10 items-start gap-2.5 rounded-xl bg-slate-50 px-2.5 py-2.5 ring-1 ring-black/[0.04] dark:bg-neutral-900/60 dark:ring-white/[0.06]"
      title={description}
    >
      <span
        className={`flex size-8 shrink-0 items-center justify-center rounded-lg ${iconClass}`}
        aria-hidden
      >
        <Icon className="size-4" strokeWidth={2} />
      </span>
      <span className="min-w-0 flex-1 pt-0.5">
        <span className="flex flex-wrap items-center gap-1.5">
          <span className="text-pretty text-sm font-medium leading-snug text-slate-700 dark:text-neutral-200">
            {label}
          </span>
          <span className="inline-flex items-center gap-1 rounded-full bg-slate-200/80 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-600 dark:bg-neutral-800 dark:text-neutral-400">
            <Lock className="size-2.5" aria-hidden />
            {badge}
          </span>
        </span>
        <span className="mt-0.5 block text-pretty text-xs leading-snug text-slate-500 dark:text-neutral-400">
          {description}
        </span>
      </span>
    </div>
  )
}

function PaletteUnavailableItem({
  label,
  description,
  badge,
  tooltip,
  icon: Icon,
  iconClass,
}: {
  label: string
  description: string
  badge: string
  tooltip: string
  icon: LucideIcon
  iconClass: string
}) {
  return (
    <div
      className="flex min-h-10 items-start gap-2.5 rounded-xl bg-slate-50/80 px-2.5 py-2.5 opacity-70 ring-1 ring-black/[0.04] dark:bg-neutral-900/40 dark:ring-white/[0.06]"
      title={tooltip}
    >
      <span
        className={`flex size-8 shrink-0 items-center justify-center rounded-lg ${iconClass}`}
        aria-hidden
      >
        <Icon className="size-4" strokeWidth={2} />
      </span>
      <span className="min-w-0 flex-1 pt-0.5">
        <span className="flex flex-wrap items-center gap-1.5">
          <span className="text-pretty text-sm font-medium leading-snug text-slate-600 dark:text-neutral-400">
            {label}
          </span>
          <span className="rounded-full bg-slate-200/80 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-500 dark:bg-neutral-800 dark:text-neutral-500">
            {badge}
          </span>
        </span>
        <span className="mt-0.5 block text-pretty text-xs leading-snug text-slate-500 dark:text-neutral-500">
          {description}
        </span>
      </span>
    </div>
  )
}

function itemMatchesQuery(
  label: string,
  description: string,
  query: string,
): boolean {
  if (!query) return true
  const haystack = `${label} ${description}`.toLowerCase()
  return haystack.includes(query)
}

export function NodePalette({
  disabled,
  codeExecutionEnabled = false,
  itemKind = 'assignment',
  onAddNode,
}: NodePaletteProps) {
  const { t } = useTranslation('common')
  const [query, setQuery] = useState('')
  const normalizedQuery = normalizeSearch(query)

  const filterItems = (items: PaletteItemConfig[]) =>
    items.filter((item) =>
      itemMatchesQuery(t(item.labelKey), t(item.descriptionKey), normalizedQuery),
    )

  const inputItems = useMemo(() => {
    const filtered = filterItems(INPUT_ITEMS)
    if (itemKind === 'quiz') {
      return filtered.filter((item) => item.type !== 'studentSubmission')
    }
    return filtered.filter((item) => item.type !== 'quizResponses')
  }, [itemKind, normalizedQuery, t])
  const processingItems = useMemo(
    () =>
      filterItems(PROCESSING_ITEMS).filter(
        (item) => item.type !== 'codeTestRunner' || codeExecutionEnabled,
      ),
    [codeExecutionEnabled, normalizedQuery, t],
  )

  const outputItems = useMemo(() => filterItems(OUTPUT_ITEMS), [normalizedQuery, t])

  const showStudentGrade =
    !normalizedQuery ||
    itemMatchesQuery(
      t('gradingAgent.canvas.palette.studentGrade'),
      t('gradingAgent.canvas.palette.description.studentGrade'),
      normalizedQuery,
    )

  const showQuizResponses =
    itemKind === 'quiz' &&
    (!normalizedQuery ||
      itemMatchesQuery(
        t('gradingAgent.canvas.palette.quizResponses'),
        t('gradingAgent.canvas.palette.description.quizResponses'),
        normalizedQuery,
      ))

  const showCodeTestsUnavailable =
    !codeExecutionEnabled &&
    itemMatchesQuery(
      t('gradingAgent.canvas.palette.codeTests'),
      t('gradingAgent.canvas.palette.description.codeTests'),
      normalizedQuery,
    )

  const hasVisibleItems =
    inputItems.length > 0 ||
    processingItems.length > 0 ||
    outputItems.length > 0 ||
    showStudentGrade ||
    showQuizResponses ||
    showCodeTestsUnavailable

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="shrink-0 space-y-2 pb-3">
        <div>
          <h2 className="text-balance text-sm font-semibold text-slate-900 dark:text-neutral-50">
            {t('gradingAgent.canvas.palette.title')}
          </h2>
          <p className="mt-1 text-pretty text-xs leading-relaxed text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.canvas.palette.hint')}
          </p>
        </div>
        <label className="relative block">
          <span className="sr-only">{t('gradingAgent.canvas.palette.searchLabel')}</span>
          <Search
            className="pointer-events-none absolute start-2.5 top-1/2 size-3.5 -translate-y-1/2 text-slate-400 dark:text-neutral-500"
            aria-hidden
          />
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t('gradingAgent.canvas.palette.searchPlaceholder')}
            className="w-full rounded-lg bg-slate-50 py-2 ps-8 pe-3 text-sm text-slate-900 shadow-sm ring-1 ring-black/[0.05] outline-none placeholder:text-slate-400 focus:ring-2 focus:ring-indigo-500/30 dark:bg-neutral-900 dark:text-neutral-100 dark:ring-white/10 dark:placeholder:text-neutral-500 dark:focus:ring-indigo-400/25"
          />
        </label>
      </div>

      <div
        className={`min-h-0 flex-1 overflow-y-auto overscroll-contain pe-0.5${disabled ? ' pointer-events-none opacity-50' : ''}`}
      >
        {hasVisibleItems ? (
          <div className="flex flex-col gap-4 pb-1">
            {inputItems.length > 0 || showQuizResponses ? (
              <PaletteGroup title={t('gradingAgent.canvas.palette.groupInput')}>
                {showQuizResponses ? (
                  <PaletteFixedItem
                    label={t('gradingAgent.canvas.palette.quizResponses')}
                    description={t('gradingAgent.canvas.palette.description.quizResponses')}
                    badge={t('gradingAgent.canvas.palette.onCanvas')}
                    icon={ListOrdered}
                    iconClass="bg-violet-500/10 text-violet-700 dark:text-violet-300"
                  />
                ) : null}
                {inputItems.map((item) => (
                  <PaletteItem
                    key={item.type}
                    type={item.type}
                    label={t(item.labelKey)}
                    description={t(item.descriptionKey)}
                    icon={item.icon}
                    iconClass={item.iconClass}
                    disabled={disabled}
                    onAddNode={onAddNode}
                  />
                ))}
              </PaletteGroup>
            ) : null}

            {processingItems.length > 0 || showCodeTestsUnavailable ? (
              <PaletteGroup title={t('gradingAgent.canvas.palette.groupProcessing')}>
                {processingItems.map((item) => (
                  <PaletteItem
                    key={item.type}
                    type={item.type}
                    label={t(item.labelKey)}
                    description={t(item.descriptionKey)}
                    icon={item.icon}
                    iconClass={item.iconClass}
                    disabled={disabled}
                    onAddNode={onAddNode}
                  />
                ))}
                {showCodeTestsUnavailable ? (
                  <PaletteUnavailableItem
                    label={t('gradingAgent.canvas.palette.codeTests')}
                    description={t('gradingAgent.canvas.palette.description.codeTests')}
                    badge={t('gradingAgent.canvas.palette.unavailable')}
                    tooltip={t('gradingAgent.canvas.palette.codeTestsDisabledTooltip')}
                    icon={Code2}
                    iconClass="bg-cyan-500/10 text-cyan-600 dark:text-cyan-400"
                  />
                ) : null}
              </PaletteGroup>
            ) : null}

            {outputItems.length > 0 || showStudentGrade ? (
              <PaletteGroup title={t('gradingAgent.canvas.palette.groupOutput')}>
                {showStudentGrade ? (
                  <PaletteFixedItem
                    label={t('gradingAgent.canvas.palette.studentGrade')}
                    description={t('gradingAgent.canvas.palette.description.studentGrade')}
                    badge={t('gradingAgent.canvas.palette.onCanvas')}
                    icon={GraduationCap}
                    iconClass="bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
                  />
                ) : null}
                {outputItems.map((item) => (
                  <PaletteItem
                    key={item.type}
                    type={item.type}
                    label={t(item.labelKey)}
                    description={t(item.descriptionKey)}
                    icon={item.icon}
                    iconClass={item.iconClass}
                    disabled={disabled}
                    onAddNode={onAddNode}
                  />
                ))}
              </PaletteGroup>
            ) : null}
          </div>
        ) : (
          <p className="px-0.5 py-6 text-center text-pretty text-sm text-slate-500 dark:text-neutral-400">
            {t('gradingAgent.canvas.palette.searchEmpty')}
          </p>
        )}
      </div>
    </div>
  )
}