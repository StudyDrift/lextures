import { describe, expect, it } from 'vitest'
import {
  dominantSourceKind,
  FACET_PRIORITY,
  sortFacetsByPriority,
  totalObservationCount,
  uniqueCourseCount,
  type EvidenceRow,
  type FacetSummary,
} from '../learner-profile-api'

describe('learner-profile-api helpers', () => {
  it('sorts facets by stable priority order', () => {
    const facets: FacetSummary[] = [
      {
        facetKey: 'interests',
        state: 'ok',
        summary: {},
        confidence: 0.8,
        computedVersion: 1,
        updatedAt: '2026-01-01T00:00:00Z',
      },
      {
        facetKey: 'study_rhythm',
        state: 'ok',
        summary: {},
        confidence: 0.7,
        computedVersion: 1,
        updatedAt: '2026-01-01T00:00:00Z',
      },
      {
        facetKey: 'learning_approach',
        state: 'insufficient_data',
        summary: {},
        confidence: 0,
        computedVersion: 1,
        updatedAt: '2026-01-01T00:00:00Z',
      },
    ]
    const sorted = sortFacetsByPriority(facets).map((f) => f.facetKey)
    expect(sorted).toEqual(['study_rhythm', 'interests', 'learning_approach'])
    expect(FACET_PRIORITY[0]).toBe('study_rhythm')
  })

  it('aggregates evidence counts and course totals', () => {
    const evidence: EvidenceRow[] = [
      {
        sourceKind: 'quiz_attempt',
        sourceTable: 'course.quiz_attempts',
        observationCount: 5,
        courseId: 'a',
      },
      {
        sourceKind: 'quiz_attempt',
        sourceTable: 'course.quiz_attempts',
        observationCount: 7,
        courseId: 'b',
      },
      {
        sourceKind: 'engagement_event',
        sourceTable: 'analytics.engagement_events',
        observationCount: 3,
      },
    ]
    expect(totalObservationCount(evidence)).toBe(15)
    expect(uniqueCourseCount(evidence)).toBe(2)
    expect(dominantSourceKind(evidence)).toBe('quiz_attempt')
  })
})