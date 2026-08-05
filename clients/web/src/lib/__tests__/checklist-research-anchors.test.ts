import { describe, expect, it } from 'vitest'
import {
  buildSourceIndex,
  courseDesignResearchHref,
  sourceToAnchorId,
} from '../checklist-research-anchors'

describe('sourceToAnchorId', () => {
  it('slugifies common source chips', () => {
    expect(sourceToAnchorId('OSCQR 7')).toBe('src-oscqr-7')
    expect(sourceToAnchorId('QM 1.2')).toBe('src-qm-1-2')
    expect(sourceToAnchorId('WCAG 1.4.3')).toBe('src-wcag-1-4-3')
    expect(sourceToAnchorId('UDL Action & Expression')).toBe('src-udl-action-and-expression')
    expect(sourceToAnchorId('ADA/§504')).toBe('src-ada-s504')
  })
})

describe('courseDesignResearchHref', () => {
  it('returns base path without source', () => {
    expect(courseDesignResearchHref()).toBe('/help/course-checklist/research')
    expect(courseDesignResearchHref(null)).toBe('/help/course-checklist/research')
  })

  it('appends source fragment', () => {
    expect(courseDesignResearchHref('OSCQR 7')).toBe('/help/course-checklist/research#src-oscqr-7')
  })
})

describe('buildSourceIndex', () => {
  it('includes known sources from the help catalog', () => {
    const index = buildSourceIndex()
    const oscqr7 = index.find((e) => e.source === 'OSCQR 7')
    expect(oscqr7).toBeDefined()
    expect(oscqr7!.anchorId).toBe('src-oscqr-7')
    expect(oscqr7!.items.length).toBeGreaterThan(0)
  })
})
