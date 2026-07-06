import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { ChevronDown, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { ConfirmDialog } from '../../confirm-dialog'
import { ImportFieldPicker } from './import-field-picker'
import { authorizedFetch } from '../../../lib/api'
import { readApiErrorMessage } from '../../../lib/errors'
import {
  fetchCourseGradingAgentTemplates,
  fetchCourseGradingAgents,
  fetchGraderAgentConfig,
  fetchGraderAgentTemplate,
  type CourseGradingAgentSummary,
  type CourseGradingAgentTemplateSummary,
  type CoursePublic,
  type GradingAgentItemKind,
} from '../../../lib/courses-api'
import { courseCatalogDisplayTitle } from '../../../pages/lms/course-catalog-display'
import type { GraderAgentWorkflowSeed } from './use-grader-agent-workflow'

type ImportSourceType = 'template' | 'agent'
type OpenImportField = 'course' | 'source' | 'item'

type ImportWorkflowMenuProps = {
  courseCode: string
  itemId: string
  itemKind: GradingAgentItemKind
  disabled?: boolean
  onImport: (seed: GraderAgentWorkflowSeed) => void
}

function courseLabel(course: CoursePublic): string {
  const title = courseCatalogDisplayTitle(course)
  return title === course.courseCode ? title : `${title} (${course.courseCode})`
}

export function ImportWorkflowMenu({
  courseCode,
  itemId,
  itemKind,
  disabled = false,
  onImport,
}: ImportWorkflowMenuProps) {
  const { t } = useTranslation('common')
  const rootRef = useRef<HTMLDivElement>(null)
  const panelId = useId()

  const [open, setOpen] = useState(false)
  const [openField, setOpenField] = useState<OpenImportField | null>(null)
  const [courses, setCourses] = useState<CoursePublic[]>([])
  const [coursesLoading, setCoursesLoading] = useState(false)
  const [sourceCourseCode, setSourceCourseCode] = useState(courseCode)
  const [sourceType, setSourceType] = useState<ImportSourceType>('template')
  const [templates, setTemplates] = useState<CourseGradingAgentTemplateSummary[]>([])
  const [agents, setAgents] = useState<CourseGradingAgentSummary[]>([])
  const [itemsLoading, setItemsLoading] = useState(false)
  const [itemsError, setItemsError] = useState<string | null>(null)
  const [selectedItemId, setSelectedItemId] = useState('')
  const [importing, setImporting] = useState(false)
  const [importError, setImportError] = useState<string | null>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [pendingImportLabel, setPendingImportLabel] = useState('')

  const sortedCourses = useMemo(
    () =>
      [...courses]
        .filter((course) => Boolean(course.courseCode))
        .sort((a, b) => courseLabel(a).localeCompare(courseLabel(b))),
    [courses],
  )

  const availableAgents = useMemo(
    () =>
      agents.filter((agent) => {
        const kind = agent.itemKind ?? 'assignment'
        if (kind !== itemKind) return false
        if (sourceCourseCode === courseCode && agent.itemId === itemId) return false
        return true
      }),
    [agents, courseCode, itemId, itemKind, sourceCourseCode],
  )

  const availableItems =
    sourceType === 'template'
      ? templates
      : availableAgents.map((agent) => ({
          id: agent.itemId,
          name: agent.assignmentTitle,
        }))

  const courseOptions = useMemo(
    () =>
      sortedCourses.map((course) => ({
        id: course.courseCode,
        label:
          course.courseCode === courseCode
            ? t('gradingAgent.import.thisCourse', { course: courseLabel(course) })
            : courseLabel(course),
      })),
    [courseCode, sortedCourses, t],
  )

  const sourceTypeOptions = useMemo(
    () => [
      { id: 'template', label: t('gradingAgent.import.sourceTemplate') },
      { id: 'agent', label: t('gradingAgent.import.sourceAgent') },
    ],
    [t],
  )

  const itemOptions = useMemo(
    () => availableItems.map((item) => ({ id: item.id, label: item.name })),
    [availableItems],
  )

  const selectedItemLabel = useMemo(() => {
    if (!selectedItemId) return ''
    if (sourceType === 'template') {
      return templates.find((template) => template.id === selectedItemId)?.name ?? ''
    }
    return availableAgents.find((agent) => agent.itemId === selectedItemId)?.assignmentTitle ?? ''
  }, [availableAgents, selectedItemId, sourceType, templates])

  useEffect(() => {
    if (!open) return
    setSourceCourseCode(courseCode)
    setSourceType('template')
    setSelectedItemId('')
    setImportError(null)
    setItemsError(null)
    setOpenField(null)
  }, [courseCode, open])

  useEffect(() => {
    if (!open) return
    let cancelled = false
    setCoursesLoading(true)
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/courses')
        const raw: unknown = await res.json().catch(() => ({}))
        if (!res.ok) throw new Error(readApiErrorMessage(raw))
        if (!cancelled) {
          setCourses(((raw as { courses?: CoursePublic[] }).courses ?? []).filter((c) => Boolean(c?.id)))
        }
      } catch (e) {
        if (!cancelled) {
          setCourses([])
          setItemsError(e instanceof Error ? e.message : t('gradingAgent.import.error.loadCourses'))
        }
      } finally {
        if (!cancelled) setCoursesLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, t])

  useEffect(() => {
    if (!open) return
    let cancelled = false
    setItemsLoading(true)
    setItemsError(null)
    void (async () => {
      try {
        if (sourceType === 'template') {
          const res = await fetchCourseGradingAgentTemplates(sourceCourseCode)
          if (!cancelled) {
            setTemplates(res.templates)
            setAgents([])
          }
        } else {
          const res = await fetchCourseGradingAgents(sourceCourseCode)
          if (!cancelled) {
            setAgents(res.agents)
            setTemplates([])
          }
        }
      } catch (e) {
        if (!cancelled) {
          setTemplates([])
          setAgents([])
          setItemsError(e instanceof Error ? e.message : t('gradingAgent.import.error.loadItems'))
        }
      } finally {
        if (!cancelled) setItemsLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, sourceCourseCode, sourceType, t])

  useEffect(() => {
    if (!open || itemsLoading) return
    if (selectedItemId && availableItems.some((item) => item.id === selectedItemId)) return
    setSelectedItemId(availableItems[0]?.id ?? '')
  }, [availableItems, itemsLoading, open, selectedItemId])

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !importing && !confirmOpen) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [confirmOpen, importing, open])

  const canImport = !disabled && !importing && !itemsLoading && selectedItemId !== ''

  const requestImport = () => {
    if (!canImport) return
    const label =
      selectedItemLabel || availableItems.find((item) => item.id === selectedItemId)?.name || ''
    if (!label) return
    setImportError(null)
    setPendingImportLabel(label)
    setConfirmOpen(true)
  }

  const performImport = async () => {
    if (!selectedItemId) return
    setImporting(true)
    setImportError(null)
    try {
      let seed: GraderAgentWorkflowSeed
      if (sourceType === 'template') {
        const { template } = await fetchGraderAgentTemplate(sourceCourseCode, selectedItemId)
        seed = {
          prompt: template.prompt,
          includeAssignmentContent: template.includeAssignmentContent,
          includeRubric: template.includeRubric,
          workflowGraph: template.workflowGraph,
        }
      } else {
        const agent = availableAgents.find((entry) => entry.itemId === selectedItemId)
        const kind = agent?.itemKind ?? 'assignment'
        const { config } = await fetchGraderAgentConfig(sourceCourseCode, selectedItemId, kind)
        if (!config) {
          throw new Error(t('gradingAgent.import.error.missingAgent'))
        }
        seed = {
          prompt: config.prompt,
          includeAssignmentContent: config.includeAssignmentContent,
          includeRubric: config.includeRubric,
          workflowGraph: config.workflowGraph,
        }
      }
      onImport(seed)
      setConfirmOpen(false)
      setOpen(false)
    } catch (e) {
      setImportError(e instanceof Error ? e.message : t('gradingAgent.import.error.import'))
    } finally {
      setImporting(false)
    }
  }

  const emptyItemsMessage =
    sourceType === 'template'
      ? t('gradingAgent.import.noTemplates')
      : t('gradingAgent.import.noAgents')

  return (
    <>
      <div ref={rootRef} className="relative shrink-0">
        <button
          type="button"
          disabled={disabled}
          aria-haspopup="dialog"
          aria-expanded={open}
          aria-controls={open ? panelId : undefined}
          onClick={() => setOpen((value) => !value)}
          className="inline-flex items-center gap-2 rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm font-semibold text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
        >
          {t('gradingAgent.import.button')}
          <ChevronDown className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`} aria-hidden />
        </button>
        {open ? (
          <div
            id={panelId}
            role="dialog"
            aria-label={t('gradingAgent.import.panelLabel')}
            className="absolute end-0 z-50 mt-1 w-[min(20rem,calc(100vw-2rem))] rounded-xl border border-slate-200 bg-white p-4 shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-900"
          >
            <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
              {t('gradingAgent.import.title')}
            </p>
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">{t('gradingAgent.import.description')}</p>

            <div className="mt-4 space-y-3">
              <ImportFieldPicker
                label={t('gradingAgent.import.courseLabel')}
                value={sourceCourseCode}
                options={courseOptions}
                disabled={importing}
                loading={coursesLoading}
                loadingLabel={t('gradingAgent.import.loadingCourses')}
                emptyLabel={t('gradingAgent.import.error.loadCourses')}
                searchable
                searchPlaceholder={t('gradingAgent.import.searchCourses')}
                noMatchLabel={t('gradingAgent.import.noMatch')}
                open={openField === 'course'}
                onOpenChange={(next) => setOpenField(next ? 'course' : null)}
                onChange={setSourceCourseCode}
              />

              <ImportFieldPicker
                label={t('gradingAgent.import.sourceTypeLabel')}
                value={sourceType}
                options={sourceTypeOptions}
                disabled={importing}
                open={openField === 'source'}
                onOpenChange={(next) => setOpenField(next ? 'source' : null)}
                onChange={(next) => setSourceType(next as ImportSourceType)}
              />

              {itemsError ? (
                <p className="text-sm text-rose-600 dark:text-rose-400" role="alert">
                  {itemsError}
                </p>
              ) : (
                <ImportFieldPicker
                  label={
                    sourceType === 'template'
                      ? t('gradingAgent.import.templateLabel')
                      : t('gradingAgent.import.agentLabel')
                  }
                  value={selectedItemId}
                  options={itemOptions}
                  disabled={importing}
                  loading={itemsLoading}
                  loadingLabel={t('gradingAgent.import.loadingItems')}
                  emptyLabel={emptyItemsMessage}
                  searchable
                  searchPlaceholder={t('gradingAgent.import.searchItems')}
                  noMatchLabel={t('gradingAgent.import.noMatch')}
                  open={openField === 'item'}
                  onOpenChange={(next) => setOpenField(next ? 'item' : null)}
                  onChange={setSelectedItemId}
                />
              )}
            </div>

            {importError ? (
              <p className="mt-3 text-sm text-rose-600 dark:text-rose-400" role="alert">
                {importError}
              </p>
            ) : null}

            <div className="mt-4 flex justify-end gap-2">
              <button
                type="button"
                disabled={importing}
                onClick={() => setOpen(false)}
                className="rounded-lg px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-100 disabled:opacity-60 dark:text-neutral-400 dark:hover:bg-neutral-800"
              >
                {t('gradingAgent.save.templateCancel')}
              </button>
              <button
                type="button"
                disabled={!canImport || availableItems.length === 0}
                onClick={requestImport}
                className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                {importing ? (
                  <>
                    <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                    {t('gradingAgent.import.importing')}
                  </>
                ) : (
                  t('gradingAgent.import.confirmButton')
                )}
              </button>
            </div>
          </div>
        ) : null}
      </div>

      <ConfirmDialog
        open={confirmOpen}
        title={t('gradingAgent.import.confirmTitle')}
        description={t('gradingAgent.import.confirmDescription', { name: pendingImportLabel })}
        confirmLabel={t('gradingAgent.import.confirmButton')}
        cancelLabel={t('gradingAgent.save.templateCancel')}
        busy={importing}
        onConfirm={() => void performImport()}
        onClose={() => {
          if (!importing) setConfirmOpen(false)
        }}
      />
    </>
  )
}