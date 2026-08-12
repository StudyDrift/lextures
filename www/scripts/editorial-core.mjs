import { readFile, readdir } from 'node:fs/promises'
import path from 'node:path'

export const PILLARS = Object.freeze([
  { id: 'p1', slug: 'adaptive-learning', title: 'Adaptive learning: how it actually works', floor: 18 },
  { id: 'p2', slug: 'assessment-design-ai', title: 'Assessment design in the age of generative AI', floor: 20 },
  { id: 'p3', slug: 'grading-and-integrity', title: 'Grading, feedback and academic integrity', floor: 16 },
  { id: 'p4', slug: 'mastery-and-standards', title: 'Standards, outcomes and mastery-based grading', floor: 14 },
  { id: 'p5', slug: 'choosing-an-lms', title: 'Choosing and running a learning platform', floor: 16 },
  { id: 'p6', slug: 'homeschool-teaching', title: 'Teaching at home: curriculum, pacing and evidence', floor: 12 },
])

export function parseFrontmatter(source) {
  const match = source.match(/^---\r?\n([\s\S]*?)\r?\n---/)
  if (!match) return {}
  return Object.fromEntries(match[1].split(/\r?\n/).flatMap(line => {
    const colon = line.indexOf(':')
    if (colon < 1 || /^\s/.test(line)) return []
    const key = line.slice(0, colon).trim()
    let value = line.slice(colon + 1).trim().replace(/^['"]|['"]$/g, '')
    if (/^\[.*\]$/.test(value)) value = value.slice(1, -1).split(',').map(item => item.trim().replace(/^['"]|['"]$/g, ''))
    return [[key, value]]
  }))
}

export function priorityScore({ intentValue, aiCitationGap, searchVolumeBand, ourCredibility, difficulty }) {
  const fields = { intentValue, aiCitationGap, searchVolumeBand, ourCredibility, difficulty }
  for (const [key, value] of Object.entries(fields)) {
    if (!Number.isFinite(value) || value < 0 || value > 5) throw new Error(`${key} must be a number from 0 to 5`)
  }
  return intentValue * 3 + aiCitationGap * 3 + searchVolumeBand * 2 + ourCredibility * 2 - difficulty
}

export function calendarRows(source) {
  return source.split(/\r?\n/).filter(line => /^\|\s*20\d\d-\d\d-\d\d\s*\|/.test(line)).map(line => {
    const cells = line.split('|').slice(1, -1).map(cell => cell.trim())
    return { date: cells[0], day: cells[1], pillar: cells[2], slug: cells[3].replace(/`/g, ''), brief: cells[4].replace(/`/g, ''), owner: cells[5], status: cells[6], notes: cells[7] }
  })
}

export async function loadEditorial(root) {
  const [calendar, blogNames, briefNames] = await Promise.all([
    readFile(path.join(root, '../docs/plan/seo/calendar.md'), 'utf8'),
    readdir(path.join(root, 'src/blog')),
    readdir(path.join(root, '../docs/plan/seo/briefs')),
  ])
  const posts = []
  for (const name of blogNames.filter(name => /\.mdx?$/.test(name))) {
    posts.push({ slug: name.replace(/\.mdx?$/, ''), ...parseFrontmatter(await readFile(path.join(root, 'src/blog', name), 'utf8')) })
  }
  const briefs = new Set(briefNames.filter(name => name !== '_TEMPLATE.md' && name.endsWith('.md')).map(name => name.replace(/\.md$/, '')))
  return { rows: calendarRows(calendar), posts, briefs }
}

export function validateEditorial({ rows, posts, briefs }, today = new Date()) {
  const errors = []
  const ids = new Set(PILLARS.map(p => p.id))
  for (const post of posts) {
    if (!ids.has(post.pillar)) errors.push(`${post.slug}: unknown or missing pillar`)
    if (!post.briefRef || !briefs.has(post.briefRef)) errors.push(`${post.slug}: briefRef does not resolve`)
    if (!/^\d{4}-\d{2}-\d{2}$/.test(post.reviewDue || '')) errors.push(`${post.slug}: reviewDue is required`)
  }
  for (const row of rows) {
    if (!ids.has(row.pillar)) errors.push(`${row.date}: unknown pillar ${row.pillar}`)
    if (row.brief && row.brief !== 'rapid-response' && !briefs.has(row.brief)) errors.push(`${row.date}: brief ${row.brief} does not resolve`)
    if (!['briefed', 'drafted', 'in review', 'published', 'missed', 'reserved'].includes(row.status)) errors.push(`${row.date}: invalid status ${row.status}`)
    if (row.status === 'missed' && !row.notes) errors.push(`${row.date}: missed slots require a reason`)
  }
  const future = rows.filter(row => new Date(`${row.date}T23:59:59Z`) >= today && row.status !== 'reserved')
  if (future.slice(0, 10).some(row => row.status !== 'briefed' || !row.owner || !row.brief)) errors.push('the next 10 publication slots must be briefed and assigned')
  return errors
}

export function gapRows({ posts, briefs }) {
  const publishedBriefs = new Set(posts.map(post => post.briefRef))
  const unpublished = [...briefs].filter(brief => !publishedBriefs.has(brief)).sort()
  const counts = new Map(PILLARS.map(p => [p.id, 0]))
  for (const post of posts) if (counts.has(post.pillar)) counts.set(post.pillar, counts.get(post.pillar) + 1)
  return { unpublished, pillars: PILLARS.map(p => ({ ...p, count: counts.get(p.id), remaining: Math.max(0, 12 - counts.get(p.id)) })) }
}

export function refreshDue(posts, today = new Date()) {
  return posts.filter(post => post.reviewDue && new Date(`${post.reviewDue}T23:59:59Z`) < today)
    .sort((a, b) => a.reviewDue.localeCompare(b.reviewDue))
}
