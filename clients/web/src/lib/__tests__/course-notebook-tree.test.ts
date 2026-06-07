import { describe, expect, it } from 'vitest'
import {
  addNotebookGroup,
  addNotebookPage,
  isNotebookGroup,
  movePageToParent,
  notebookGroupMoveTargets,
  reorderAmongSiblings,
  reparentPage,
  type CourseNotebookPage,
} from '../course-notebook-tree'

function pagesSeed(): CourseNotebookPage[] {
  return [
    { id: 'a', title: 'A', parentId: null, sortOrder: 0, kind: 'page', contentMd: '' },
    { id: 'b', title: 'B', parentId: null, sortOrder: 1, kind: 'page', contentMd: '' },
    { id: 'c', title: 'C', parentId: null, sortOrder: 2, kind: 'page', contentMd: '' },
  ]
}

describe('reorderAmongSiblings', () => {
  it('moves root pages', () => {
    const next = reorderAmongSiblings(pagesSeed(), null, 'c', 'a')
    const order = next.filter((p) => p.parentId === null).sort((x, y) => x.sortOrder - y.sortOrder)
    expect(order.map((p) => p.id)).toEqual(['c', 'a', 'b'])
  })
})

describe('reparentPage', () => {
  it('moves a page under another parent', () => {
    const pages: CourseNotebookPage[] = [
      { id: 'a', title: 'A', parentId: null, sortOrder: 0, kind: 'page', contentMd: '' },
      { id: 'b', title: 'B', parentId: null, sortOrder: 1, kind: 'page', contentMd: '' },
    ]
    const next = reparentPage(pages, 'b', 'a', 'a')
    expect(next).not.toBeNull()
    const b = next!.find((p) => p.id === 'b')
    expect(b?.parentId).toBe('a')
  })

  it('rejects nesting into own descendant', () => {
    const pages: CourseNotebookPage[] = [
      { id: 'a', title: 'A', parentId: null, sortOrder: 0, kind: 'page', contentMd: '' },
      { id: 'b', title: 'B', parentId: 'a', sortOrder: 0, kind: 'page', contentMd: '' },
    ]
    expect(reparentPage(pages, 'a', 'b', null)).toBeNull()
  })
})

describe('addNotebookGroup', () => {
  it('creates a group page', () => {
    const { pages, newId } = addNotebookGroup(pagesSeed(), null, 'Week 1')
    const group = pages.find((p) => p.id === newId)
    expect(group).toMatchObject({ title: 'Week 1', kind: 'group', parentId: null })
    expect(isNotebookGroup(group!)).toBe(true)
  })

  it('creates nested pages under a group', () => {
    const { pages: withGroup, newId: groupId } = addNotebookGroup([], null, 'Topics')
    const { pages, newId } = addNotebookPage(withGroup, groupId, 'Notes')
    const page = pages.find((p) => p.id === newId)
    expect(page?.parentId).toBe(groupId)
    expect(page?.kind).toBe('page')
  })
})

describe('movePageToParent', () => {
  it('moves a root page into a group', () => {
    const pages: CourseNotebookPage[] = [
      { id: 'g', title: 'Group', parentId: null, sortOrder: 0, kind: 'group', contentMd: '' },
      { id: 'p', title: 'Page', parentId: null, sortOrder: 1, kind: 'page', contentMd: '' },
    ]
    const next = movePageToParent(pages, 'p', 'g')
    expect(next?.find((p) => p.id === 'p')?.parentId).toBe('g')
  })

  it('moves a nested page into another group', () => {
    const pages: CourseNotebookPage[] = [
      { id: 'a', title: 'A', parentId: null, sortOrder: 0, kind: 'group', contentMd: '' },
      { id: 'b', title: 'B', parentId: null, sortOrder: 1, kind: 'group', contentMd: '' },
      { id: 'p', title: 'P', parentId: 'a', sortOrder: 0, kind: 'page', contentMd: '' },
    ]
    const next = movePageToParent(pages, 'p', 'b')
    expect(next?.find((p) => p.id === 'p')?.parentId).toBe('b')
  })
})

describe('notebookGroupMoveTargets', () => {
  it('excludes self and descendant groups', () => {
    const pages: CourseNotebookPage[] = [
      { id: 'g', title: 'G', parentId: null, sortOrder: 0, kind: 'group', contentMd: '' },
      { id: 'sg', title: 'SG', parentId: 'g', sortOrder: 0, kind: 'group', contentMd: '' },
    ]
    expect(notebookGroupMoveTargets(pages, 'g').map((p) => p.id)).toEqual([])
    expect(notebookGroupMoveTargets(pages, 'sg').map((p) => p.id)).toEqual(['g'])
  })
})
