import { useEffect, useId, useState } from 'react'
import { ChevronDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatDate } from '../../../lib/format'
import {
  fetchLearnerProfileFacet,
  type FacetKey,
  type FacetSummary,
  type Insight,
} from '../../../lib/learner-profile-api'
import { SettingsSection } from '../settings-section'
import { ConfidenceIndicator } from './confidence-indicator'
import { FacetVisualization } from './facet-visualization'
import { InsightRow } from './insight-row'

const FACET_META: Record<
  FacetKey,
  { titleKey: string; descriptionKey: string; defaultCollapsed?: boolean }
> = {
  study_rhythm: {
    titleKey: 'learnerProfile.facet.studyRhythm.title',
    descriptionKey: 'learnerProfile.facet.studyRhythm.description',
  },
  content_modality: {
    titleKey: 'learnerProfile.facet.contentModality.title',
    descriptionKey: 'learnerProfile.facet.contentModality.description',
  },
  strengths_growth: {
    titleKey: 'learnerProfile.facet.strengthsGrowth.title',
    descriptionKey: 'learnerProfile.facet.strengthsGrowth.description',
  },
  interests: {
    titleKey: 'learnerProfile.facet.interests.title',
    descriptionKey: 'learnerProfile.facet.interests.description',
  },
  learning_approach: {
    titleKey: 'learnerProfile.facet.learningApproach.title',
    descriptionKey: 'learnerProfile.facet.learningApproach.description',
    defaultCollapsed: true,
  },
}

type Props = {
  facet: FacetSummary
}

export function FacetSection({ facet }: Props) {
  const { t } = useTranslation('learnerProfile')
  const contentId = useId()
  const meta = FACET_META[facet.facetKey]
  const [collapsed, setCollapsed] = useState(meta.defaultCollapsed === true)
  const [insights, setInsights] = useState<Insight[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (facet.state !== 'ok') return
    let cancelled = false
    ;(async () => {
      setLoading(true)
      setError(null)
      try {
        const detail = await fetchLearnerProfileFacet(facet.facetKey)
        if (cancelled) return
        setInsights(detail?.insights ?? [])
      } catch {
        if (!cancelled) {
          setError(t('learnerProfile.facet.error'))
          setInsights([])
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [facet.facetKey, facet.state, t])

  const body = () => {
    if (facet.state === 'insufficient_data') {
      return (
        <p className="text-sm text-slate-500 dark:text-neutral-400">
          {t('learnerProfile.facet.insufficient')}
        </p>
      )
    }
    if (error) {
      return (
        <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">
          {error}
        </p>
      )
    }
    if (loading) {
      return (
        <p className="text-sm text-slate-500 dark:text-neutral-400">
          {t('learnerProfile.facet.loading')}
        </p>
      )
    }
    return (
      <div className="space-y-3">
        <FacetVisualization facetKey={facet.facetKey} summary={facet.summary} />
        {(insights ?? []).map((insight) => (
          <InsightRow key={insight.insightKey} facetKey={facet.facetKey} insight={insight} />
        ))}
      </div>
    )
  }

  return (
    <SettingsSection
      id={`facet-${facet.facetKey}`}
      title={t(meta.titleKey)}
      description={t(meta.descriptionKey)}
    >
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <button
          type="button"
          className="inline-flex items-center gap-1.5 text-sm font-medium text-slate-700 hover:text-slate-900 dark:text-neutral-200 dark:hover:text-white"
          aria-expanded={!collapsed}
          aria-controls={contentId}
          onClick={() => setCollapsed((c) => !c)}
        >
          <ChevronDown
            className={`h-4 w-4 transition-transform ${collapsed ? '' : 'rotate-180'}`}
            aria-hidden
          />
          {collapsed ? t('learnerProfile.facet.expand') : t('learnerProfile.facet.collapse')}
        </button>
        <div className="flex flex-wrap items-center gap-3 text-xs text-slate-500 dark:text-neutral-400">
          <span>
            {t('learnerProfile.facet.lastComputed', {
              date: formatDate(facet.updatedAt, { dateStyle: 'medium', timeStyle: 'short' }),
            })}
          </span>
          <ConfidenceIndicator score={facet.confidence} />
        </div>
      </div>
      {!collapsed ? <div id={contentId}>{body()}</div> : null}
    </SettingsSection>
  )
}