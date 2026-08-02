import { describe, expect, it } from 'vitest'
import {
  courseExportImportStatLines,
  summarizeCourseExportBundle,
} from '../course-export-import-stats'

describe('summarizeCourseExportBundle', () => {
  it('counts structure kinds and roster', () => {
    const stats = summarizeCourseExportBundle({
      formatVersion: 1,
      courseCode: 'C-SRC',
      course: { title: 'Biology 101' },
      syllabus: [{ id: 'a' }, { id: 'b' }],
      grading: {
        gradingScale: 'percent',
        assignmentGroups: [{ id: '1', name: 'HW' }, { id: '2', name: 'Exams' }],
      },
      structure: [
        { id: 'm1', kind: 'module', title: 'M1' },
        { id: 'h1', kind: 'heading', title: 'H', parentId: 'm1' },
        { id: 'p1', kind: 'content_page', title: 'P', parentId: 'm1' },
        { id: 'a1', kind: 'assignment', title: 'A', parentId: 'm1' },
        { id: 'a2', kind: 'assignment', title: 'A2', parentId: 'm1' },
        { id: 'q1', kind: 'quiz', title: 'Q', parentId: 'm1' },
        { id: 'e1', kind: 'external_link', title: 'L', parentId: 'm1' },
      ],
      contentPages: { p1: { markdown: '' } },
      assignments: { a1: {}, a2: {} },
      quizzes: { q1: {} },
      enrollments: [
        { email: 'a@example.com', role: 'student' },
        { email: 'b@example.com', role: 'student' },
      ],
    })

    expect(stats.sourceCourseCode).toBe('C-SRC')
    expect(stats.title).toBe('Biology 101')
    expect(stats.modules).toBe(1)
    expect(stats.headings).toBe(1)
    expect(stats.contentPages).toBe(1)
    expect(stats.assignments).toBe(2)
    expect(stats.quizzes).toBe(1)
    expect(stats.externalLinks).toBe(1)
    expect(stats.syllabusSections).toBe(2)
    expect(stats.assignmentGroups).toBe(2)
    expect(stats.enrollments).toBe(2)
    expect(stats.contentToolInstances).toBe(0)
    expect(stats.hasCourseSettings).toBe(true)

    const lines = courseExportImportStatLines(stats)
    expect(lines.find((l) => l.key === 'assignments')?.count).toBe(2)
    expect(lines.find((l) => l.key === 'enrollments')?.count).toBe(2)
    expect(lines.every((l) => l.count > 0)).toBe(true)
  })

  it('counts content tool instances', () => {
    const stats = summarizeCourseExportBundle({
      formatVersion: 1,
      courseCode: 'C-CT',
      structure: [],
      contentToolInstances: [
        { id: 'i1', toolId: 'flashcards' },
        { id: 'i2', toolId: 'drag_drop' },
      ],
    })
    expect(stats.contentToolInstances).toBe(2)
    expect(courseExportImportStatLines(stats).find((l) => l.key === 'contentToolInstances')?.count).toBe(
      2,
    )
  })

  it('falls back to body maps when structure is empty', () => {
    const stats = summarizeCourseExportBundle({
      formatVersion: 1,
      courseCode: 'C-X',
      structure: [],
      contentPages: { a: {}, b: {} },
      assignments: { x: {} },
      quizzes: {},
      enrollments: [],
    })
    expect(stats.contentPages).toBe(2)
    expect(stats.assignments).toBe(1)
    expect(stats.quizzes).toBe(0)
    expect(stats.contentToolInstances).toBe(0)
  })

  it('rejects non-objects', () => {
    expect(() => summarizeCourseExportBundle(null)).toThrow(/JSON object/)
    expect(() => summarizeCourseExportBundle([])).toThrow(/JSON object/)
  })
})
