import { describe, expect, it } from 'vitest'
import type { ModuleAssignmentSubmissionApi } from '../../../lib/courses-api'
import {
  adjacentSubmissionIndex,
  adjacentUngradedSubmissionIndex,
  defaultSubmissionIndex,
  sortSubmissionsByStudentLabel,
  submissionNavigatorKey,
  submissionsMatch,
} from '../submission-navigator-utils'

function submission(
  id: string,
  label: string,
  isGraded?: boolean,
): ModuleAssignmentSubmissionApi {
  return {
    id,
    submittedByDisplayName: label,
    attachmentFileId: null,
    submittedAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    isGraded,
  }
}

describe('defaultSubmissionIndex', () => {
  it('returns 0 for an empty list', () => {
    expect(defaultSubmissionIndex([])).toBe(0)
  })

  it('selects the first ungraded submission in sorted order', () => {
    const list = sortSubmissionsByStudentLabel([
      submission('a', 'Alice', true),
      submission('b', 'Bob', false),
      submission('c', 'Carol', false),
    ])
    expect(defaultSubmissionIndex(list)).toBe(1)
  })

  it('falls back to 0 when every submission is graded', () => {
    const list = sortSubmissionsByStudentLabel([
      submission('a', 'Alice', true),
      submission('b', 'Bob', true),
    ])
    expect(defaultSubmissionIndex(list)).toBe(0)
  })

  it('skips students without a submission and selects the first submitted, ungraded row', () => {
    const list = sortSubmissionsByStudentLabel([
      { submittedBy: 'user-1', submittedByDisplayName: 'Alice', attachmentFileId: null, isGraded: false },
      submission('b', 'Bob', false),
      { submittedBy: 'user-3', submittedByDisplayName: 'Carol', attachmentFileId: null, isGraded: false },
    ])
    expect(defaultSubmissionIndex(list)).toBe(1)
  })

  it('falls back to the first submitted student when every submission is graded', () => {
    const list = sortSubmissionsByStudentLabel([
      { submittedBy: 'user-1', submittedByDisplayName: 'Alice', attachmentFileId: null, isGraded: false },
      submission('b', 'Bob', true),
      submission('c', 'Carol', true),
    ])
    expect(defaultSubmissionIndex(list)).toBe(1)
  })
})

describe('submissionNavigatorKey', () => {
  it('falls back to submittedBy when there is no submission id', () => {
    expect(
      submissionNavigatorKey(
        {
          submittedBy: 'user-1',
          attachmentFileId: null,
        },
        2,
      ),
    ).toBe('user-1')
  })
})

describe('adjacentUngradedSubmissionIndex', () => {
  it('skips graded and missing students when moving forward', () => {
    const list = sortSubmissionsByStudentLabel([
      submission('a', 'Alice', true),
      { submittedBy: 'user-2', submittedByDisplayName: 'Bob', attachmentFileId: null, isGraded: false },
      submission('c', 'Carol', false),
      submission('d', 'Dan', false),
    ])
    expect(adjacentUngradedSubmissionIndex(list, 0, 1)).toBe(2)
    expect(adjacentUngradedSubmissionIndex(list, 2, 1)).toBe(3)
    expect(adjacentUngradedSubmissionIndex(list, 3, 1)).toBeNull()
  })

  it('skips graded and missing students when moving backward', () => {
    const list = sortSubmissionsByStudentLabel([
      submission('a', 'Alice', false),
      submission('b', 'Bob', true),
      submission('c', 'Carol', false),
    ])
    expect(adjacentUngradedSubmissionIndex(list, 2, -1)).toBe(0)
    expect(adjacentUngradedSubmissionIndex(list, 0, -1)).toBeNull()
  })
})

describe('adjacentSubmissionIndex', () => {
  it('returns the next or previous roster index', () => {
    const list = sortSubmissionsByStudentLabel([
      submission('a', 'Alice'),
      submission('b', 'Bob'),
      submission('c', 'Carol'),
    ])
    expect(adjacentSubmissionIndex(list, 1, 1)).toBe(2)
    expect(adjacentSubmissionIndex(list, 1, -1)).toBe(0)
    expect(adjacentSubmissionIndex(list, 0, -1)).toBeNull()
    expect(adjacentSubmissionIndex(list, 2, 1)).toBeNull()
  })
})

describe('submissionsMatch', () => {
  it('matches roster rows by submittedBy when ids are missing', () => {
    const row = {
      submittedBy: 'user-1',
      attachmentFileId: null,
    }
    expect(submissionsMatch(row, { ...row })).toBe(true)
    expect(submissionsMatch(row, { submittedBy: 'user-2', attachmentFileId: null })).toBe(false)
  })
})