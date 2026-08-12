import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..')

test('perf-budget.json has content and interactive classes with required metrics', async () => {
  const budget = JSON.parse(await readFile(path.join(ROOT, 'perf-budget.json'), 'utf8'))
  assert.ok(budget.classes.content)
  assert.ok(budget.classes.interactive)
  for (const cls of ['content', 'interactive']) {
    const c = budget.classes[cls]
    assert.equal(typeof c.jsGzipKb, 'number')
    assert.equal(typeof c.cssGzipKb, 'number')
    assert.equal(typeof c.totalGzipKb, 'number')
    assert.equal(typeof c.labLcpMs, 'number')
    assert.equal(typeof c.cls, 'number')
  }
  assert.ok(Array.isArray(budget.benchmarkUrls))
  assert.ok(budget.benchmarkUrls.includes('/privacy'))
  assert.ok(budget.benchmarkUrls.includes('/pricing/calculator'))
})

test('content class is tighter than interactive for JS', async () => {
  const budget = JSON.parse(await readFile(path.join(ROOT, 'perf-budget.json'), 'utf8'))
  assert.ok(budget.classes.content.jsGzipKb < budget.classes.interactive.jsGzipKb)
})
