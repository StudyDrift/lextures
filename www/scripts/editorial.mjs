#!/usr/bin/env node
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { gapRows, loadEditorial, refreshDue, validateEditorial } from './editorial-core.mjs'

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const command = process.argv[2] || 'calendar'
const data = await loadEditorial(ROOT)
const errors = validateEditorial(data)
if (errors.length) {
  console.error(errors.map(error => `ERROR ${error}`).join('\n'))
  process.exitCode = 1
}

if (command === 'calendar') {
  console.table(data.rows.map(({ date, pillar, slug, owner, status }) => ({ date, pillar, slug, owner, status })))
} else if (command === 'gaps') {
  const gaps = gapRows(data)
  console.log('Briefed but unpublished:')
  for (const brief of gaps.unpublished) console.log(`- ${brief}`)
  console.log('\nPillar floor (12 published articles):')
  for (const pillar of gaps.pillars) console.log(`- ${pillar.id} ${pillar.title}: ${pillar.count}/12 (${pillar.remaining} remaining)`)
} else if (command === 'refresh-due') {
  const due = refreshDue(data.posts)
  if (!due.length) console.log('No articles are past reviewDue.')
  for (const post of due) console.log(`${post.reviewDue}  ${post.slug}`)
} else {
  console.error(`Unknown command: ${command}`)
  process.exitCode = 1
}
