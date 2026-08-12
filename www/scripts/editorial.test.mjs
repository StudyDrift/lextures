import test from 'node:test'
import assert from 'node:assert/strict'
import { calendarRows, gapRows, priorityScore, refreshDue, validateEditorial } from './editorial-core.mjs'

test('priority score follows the documented formula and validates bands', () => {
  assert.equal(priorityScore({ intentValue: 4, aiCitationGap: 5, searchVolumeBand: 3, ourCredibility: 4, difficulty: 2 }), 39)
  assert.throws(() => priorityScore({ intentValue: 6, aiCitationGap: 1, searchVolumeBand: 1, ourCredibility: 1, difficulty: 1 }))
})

test('calendar rows parse operational fields', () => {
  const rows = calendarRows('| 2026-09-01 | Tue | p2 | `rubric-design` | `2026-09-rubric-design` | Chase Willden | briefed | — |')
  assert.deepEqual(rows[0], { date: '2026-09-01', day: 'Tue', pillar: 'p2', slug: 'rubric-design', brief: '2026-09-rubric-design', owner: 'Chase Willden', status: 'briefed', notes: '—' })
})

test('validation connects articles, briefs, pillars, and review dates', () => {
  const data = { rows: [], posts: [{ slug: 'x', pillar: 'p2', briefRef: 'brief-x', reviewDue: '2027-01-01' }], briefs: new Set(['brief-x']) }
  assert.deepEqual(validateEditorial(data, new Date('2028-01-01')), [])
  assert.deepEqual(refreshDue(data.posts, new Date('2028-01-01')).map(post => post.slug), ['x'])
  assert.equal(gapRows(data).pillars.find(p => p.id === 'p2').count, 1)
})
