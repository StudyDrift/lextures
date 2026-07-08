import { useCallback, useId, useState } from 'react'
import { ChevronDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatDate } from '../../../lib/format'
import {
  fetchLearnerProfileFacetEvidence,
  totalObservationCount,
  uniqueCourseCount,
  type EvidenceRow,
  type FacetKey,
  type Insight,
} from '../../../lib/learner-profile-api'
import { formatInsightValue } from '../../../lib/learner-profile-format'
import { recordLearnerProfileEvidenceExpanded } from '../../../lib/learner-profile-observability'
import { ConfidenceIndicator } from './confidence-indicator'
import { insightLabelKey } from './insight-label'

type Props = {
  facetKey: FacetKey
  insight: Insight
}

function sourceLabel(t: ReturnType<typeof useTranslation>['t'], kind: string): string {
  const key = `learnerProfile.evidence.source.${kind}`
  const translated = t(key)
  return translated === key ? t('learnerProfile.evidence.source.generic') : translated
}

function formatWindow(start?: string, end?: string): string {
  if (!start && !end) return '—'
  const a = start ? formatDate(start, { dateStyle: 'medium' }) : '…'
  const b = end ? formatDate(end, { dateStyle: 'medium' }) : '…'
  return `${a} – ${b}`
}

export function InsightRow({ facetKey, insight }: Props) {
  const { t } = useTranslation('learnerProfile')
  const panelId = useId()
  const [expanded, setExpanded] = useState(false)
  const [evidence, setEvidence] = useState<EvidenceRow[] | null>(null)
  const [loadingEvidence, setLoadingEvidence] = useState(false)
  const [evidenceError, setEvidenceError] = useState<string | null>(null)

  const derivedCount = evidence ? totalObservationCount(evidence) : 0
  const derivedCourses = evidence ? uniqueCourseCount(evidence) : 0

  const loadEvidence = useCallback(async () => {
    if (evidence !== null || loadingEvidence) return
    setLoadingEvidence(true)
    setEvidenceError(null)
    try {
      const map = await fetchLearnerProfileFacetEvidence(facetKey)
      setEvidence(map[insight.insightKey] ?? [])
      recordLearnerProfileEvidenceExpanded(facetKey, insight.insightKey)
    } catch {
      setEvidenceError(t('learnerProfile.evidence.error'))
    } finally {
      setLoadingEvidence(false)
    }
  }, [evidence, facetKey, insight.insightKey, loadingEvidence, t])

  const toggle = () => {
    const next = !expanded
    setExpanded(next)
    if (next) void loadEvidence()
  }

  const summaryLine =
    evidence && evidence.length > 0
      ? t('learnerProfile.evidence.derivedFrom', {
          count: derivedCount,
          courses: derivedCourses,
        })
      : t('learnerProfile.evidence.derivedFrom', {
          count: insight.evidence ? totalObservationCount(insight.evidence) : 0,
          courses: insight.evidence ? uniqueCourseCount(insight.evidence) : 0,
        })

  return (
    <article className="rounded-xl border border-slate-100 bg-slate-50/80 p-4 dark:border-neutral-800 dark:bg-neutral-800/40">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <h4 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            {t(insightLabelKey(insight.insightKey))}
          </h4>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
            {formatInsightValue(t, insight, facetKey)}
          </p>
          <p className="mt-2 text-xs text-slate-500 dark:text-neutral-400">{summaryLine}</p>
        </div>
        <ConfidenceIndicator score={insight.confidence} />
      </div>

      <button
        type="button"
        className="mt-3 inline-flex items-center gap-1.5 text-sm font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-300 dark:hover:text-indigo-200"
        aria-expanded={expanded}
        aria-controls={panelId}
        onClick={toggle}
      >
        <ChevronDown
          className={`h-4 w-4 transition-transform ${expanded ? 'rotate-180' : ''}`}
          aria-hidden
        />
        {expanded ? t('learnerProfile.evidence.collapse') : t('learnerProfile.evidence.why')}
      </button>

      {expanded ? (
        <div id={panelId} className="mt-3 border-t border-slate-200 pt-3 dark:border-neutral-700">
          {loadingEvidence ? (
            <p className="text-sm text-slate-500 dark:text-neutral-400">
              {t('learnerProfile.evidence.loading')}
            </p>
          ) : null}
          {evidenceError ? (
            <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">
              {evidenceError}
            </p>
          ) : null}
          {!loadingEvidence && !evidenceError && evidence && evidence.length === 0 ? (
            <p className="text-sm text-slate-500 dark:text-neutral-400">
              {t('learnerProfile.evidence.empty')}
            </p>
          ) : null}
          {!loadingEvidence && evidence && evidence.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="min-w-full text-start text-xs text-slate-700 dark:text-neutral-200">
                <caption className="sr-only">{t('learnerProfile.evidence.table.caption')}</caption>
                <thead>
                  <tr className="border-b border-slate-200 dark:border-neutral-700">
                    <th scope="col" className="px-2 py-1.5 font-semibold">
                      {t('learnerProfile.evidence.table.source')}
                    </th>
                    <th scope="col" className="px-2 py-1.5 font-semibold">
                      {t('learnerProfile.evidence.table.count')}
                    </th>
                    <th scope="col" className="px-2 py-1.5 font-semibold">
                      {t('learnerProfile.evidence.table.courses')}
                    </th>
                    <th scope="col" className="px-2 py-1.5 font-semibold">
                      {t('learnerProfile.evidence.table.window')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {evidence.map((row, index) => (
                    <tr
                      key={`${row.sourceKind}-${row.sourceTable}-${index}`}
                      className="border-b border-slate-100 dark:border-neutral-800"
                    >
                      <td className="px-2 py-1.5">{sourceLabel(t, row.sourceKind)}</td>
                      <td className="px-2 py-1.5">{row.observationCount}</td>
                      <td className="px-2 py-1.5">{row.courseId ? 1 : '—'}</td>
                      <td className="px-2 py-1.5">
                        {formatWindow(row.windowStart, row.windowEnd)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}
        </div>
      ) : null}
    </article>
  )
}