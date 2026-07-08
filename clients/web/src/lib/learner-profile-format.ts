import type { TFunction } from 'i18next'
import type { FacetKey, Insight } from './learner-profile-api'

export type ConfidenceLevel = 'high' | 'medium' | 'low'

export function confidenceLevel(score: number): ConfidenceLevel {
  if (score >= 0.75) return 'high'
  if (score >= 0.45) return 'medium'
  return 'low'
}

export function formatInsightValue(t: TFunction, insight: Insight, _facetKey: FacetKey): string {
  const value = insight.value ?? {}
  switch (insight.insightKey) {
    case 'peak_study_window': {
      const windows = value.peakWindows as Array<{ dow?: string; hourBucket?: string; share?: number }> | undefined
      const top = windows?.[0]
      if (!top?.dow || !top?.hourBucket) {
        return t('learnerProfile.insight.studyRhythm.peakUnknown')
      }
      return t('learnerProfile.insight.studyRhythm.peak', {
        day: top.dow,
        hour: top.hourBucket,
        share: Math.round((top.share ?? 0) * 100),
      })
    }
    case 'study_consistency':
      return t('learnerProfile.insight.studyRhythm.consistency', {
        score: Math.round(((value.consistencyScore as number) ?? 0) * 100),
        daysPerWeek: ((value.activeDaysPerWeek as number) ?? 0).toFixed(1),
      })
    case 'study_streak':
      return t('learnerProfile.insight.studyRhythm.streak', {
        current: value.currentStreakDays ?? 0,
        longest: value.longestStreakDays ?? 0,
      })
    case 'session_shape':
      return t('learnerProfile.insight.studyRhythm.session', {
        minutes: value.medianSessionMin ?? 0,
        perWeek: ((value.sessionsPerActiveWeek as number) ?? 0).toFixed(1),
      })
    case 'modality_affinity': {
      const affinity = value.modalityAffinity as Record<string, number> | undefined
      if (!affinity) return t('learnerProfile.insight.genericUnknown')
      const entries = Object.entries(affinity).sort((a, b) => b[1] - a[1])
      const top = entries[0]
      if (!top) return t('learnerProfile.insight.genericUnknown')
      return t('learnerProfile.insight.modality.top', {
        modality: top[0],
        share: Math.round(top[1] * 100),
      })
    }
    case 'complexity_comfort': {
      const band = value.complexityComfort as { low?: string; high?: string } | undefined
      if (!band?.low || !band?.high) return t('learnerProfile.insight.genericUnknown')
      return t('learnerProfile.insight.modality.comfort', { low: band.low, high: band.high })
    }
    case 'content_pacing':
      return t('learnerProfile.insight.modality.pacing', {
        pacing: String(value.pacing ?? 'unknown'),
      })
    case 'top_strengths': {
      const strengths = value.strengths as Array<{ concept?: string }> | undefined
      const names = (strengths ?? []).slice(0, 3).map((s) => s.concept).filter(Boolean)
      if (names.length === 0) return t('learnerProfile.insight.genericUnknown')
      return t('learnerProfile.insight.strengths.list', { concepts: names.join(', ') })
    }
    case 'growth_areas': {
      const growth = value.growth as Array<{ concept?: string; misconception?: string }> | undefined
      const names = (growth ?? [])
        .slice(0, 3)
        .map((g) => g.concept ?? g.misconception)
        .filter(Boolean)
      if (names.length === 0) return t('learnerProfile.insight.genericUnknown')
      return t('learnerProfile.insight.growth.list', { concepts: names.join(', ') })
    }
    case 'needs_review': {
      const items = value.needsReview as Array<{ concept?: string }> | undefined
      const names = (items ?? []).slice(0, 3).map((i) => i.concept).filter(Boolean)
      if (names.length === 0) return t('learnerProfile.insight.genericUnknown')
      return t('learnerProfile.insight.review.list', { concepts: names.join(', ') })
    }
    case 'persistence':
      return t('learnerProfile.insight.approach.persistence', {
        level: String((value.level as string) ?? 'unknown'),
        productive: value.productive === true,
      })
    case 'help_seeking':
      return t('learnerProfile.insight.approach.helpSeeking', {
        style: String((value.style as string) ?? 'unknown'),
        hints: ((value.hintsPerAttempt as number) ?? 0).toFixed(1),
      })
    case 'consolidation':
      return t('learnerProfile.insight.approach.consolidation', {
        level: String((value.level as string) ?? 'unknown'),
        actions: value.notebookActions ?? 0,
      })
    default:
      if (insight.insightKey.startsWith('topic_')) {
        const topic = value.topic as string | undefined
        if (!topic) return t('learnerProfile.insight.genericUnknown')
        return t('learnerProfile.insight.interests.top', {
          topic,
          affinity: Math.round(((value.affinity as number) ?? 0) * 100),
        })
      }
      return insight.label || t('learnerProfile.insight.genericUnknown')
  }
}