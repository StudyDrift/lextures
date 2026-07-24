import { describe, expect, it } from 'vitest'
import { structureReorderDropAction } from '../course-modules-reorder'

describe('structureReorderDropAction', () => {
  it('noops without a course code', () => {
    expect(
      structureReorderDropAction({
        hasCourseCode: false,
        overId: 'b',
        activeId: 'a',
        committedDuringDrag: true,
      }),
    ).toBe('noop')
  })

  it('reverts when dropped outside after a live reorder', () => {
    expect(
      structureReorderDropAction({
        hasCourseCode: true,
        overId: null,
        activeId: 'a',
        committedDuringDrag: true,
      }),
    ).toBe('revert')
  })

  it('persists when drop target equals active after live reorder', () => {
    expect(
      structureReorderDropAction({
        hasCourseCode: true,
        overId: 'a',
        activeId: 'a',
        committedDuringDrag: true,
      }),
    ).toBe('persist-current')
  })

  it('noops when pick-up and drop without moving', () => {
    expect(
      structureReorderDropAction({
        hasCourseCode: true,
        overId: 'a',
        activeId: 'a',
        committedDuringDrag: false,
      }),
    ).toBe('noop')
  })

  it('applies over when active and over differ', () => {
    expect(
      structureReorderDropAction({
        hasCourseCode: true,
        overId: 'b',
        activeId: 'a',
        committedDuringDrag: false,
      }),
    ).toBe('apply-over')
  })
})
