import { describe, expect, it } from 'vitest'
import {
  featureDefaultOn,
  mergeCoursesWithNavFeatures,
  normalizeSearchCourseItem,
} from '../search-course-features'

describe('featureDefaultOn', () => {
  it('treats undefined as enabled for opt-out features', () => {
    expect(featureDefaultOn(undefined)).toBe(true)
    expect(featureDefaultOn(true)).toBe(true)
    expect(featureDefaultOn(false)).toBe(false)
  })
})

describe('normalizeSearchCourseItem', () => {
  it('defaults live sessions off when the search index omits the flag', () => {
    expect(normalizeSearchCourseItem({ courseCode: 'X', title: 'Y' }).liveSessionsEnabled).toBe(
      false,
    )
  })
})

describe('mergeCoursesWithNavFeatures', () => {
  it('overlays nav flags for the active course', () => {
    const merged = mergeCoursesWithNavFeatures(
      [{ courseCode: 'BIO', title: 'Biology', liveSessionsEnabled: false }],
      'BIO',
      {
        notebookEnabled: true,
        feedEnabled: true,
        calendarEnabled: true,
        questionBankEnabled: false,
        standardsAlignmentEnabled: false,
        discussionsEnabled: false,
        collabDocsEnabled: false,
        sbgEnabled: false,
        liveSessionsEnabled: true,
        groupSpacesEnabled: false,
        officeHoursEnabled: false,
        filesEnabled: true,
        attendanceEnabled: false,
        whiteboardEnabled: false,
        reportCardsEnabled: false,
        visualBoardsEnabled: false,
        interactiveQuizzesEnabled: false,
      },
    )
    const bio = merged.find((c) => c.courseCode === 'BIO')
    expect(bio?.liveSessionsEnabled).toBe(true)
  })

  it('adds a synthetic course row when the active course is missing from the index', () => {
    const merged = mergeCoursesWithNavFeatures([], 'NEW', {
      notebookEnabled: true,
      feedEnabled: true,
      calendarEnabled: true,
      questionBankEnabled: false,
      standardsAlignmentEnabled: false,
      discussionsEnabled: false,
      collabDocsEnabled: false,
      sbgEnabled: false,
      liveSessionsEnabled: false,
      groupSpacesEnabled: false,
      officeHoursEnabled: false,
      filesEnabled: true,
      attendanceEnabled: false,
      whiteboardEnabled: false,
      reportCardsEnabled: false,
      visualBoardsEnabled: false,
      interactiveQuizzesEnabled: false,
    })
    expect(merged).toHaveLength(1)
    expect(merged[0]?.courseCode).toBe('NEW')
    expect(merged[0]?.liveSessionsEnabled).toBe(false)
  })
})
