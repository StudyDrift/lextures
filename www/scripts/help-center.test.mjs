import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { readdir, readFile } from 'node:fs/promises'

const docsRoot = new URL('../src/docs/', import.meta.url)

describe('categorized help center', () => {
  it('publishes at least 60 articles across at least 14 categories', async () => {
    const categories = (await readdir(docsRoot, { withFileTypes: true })).filter(entry => entry.isDirectory())
    const files = (await Promise.all(categories.map(entry => readdir(new URL(`${entry.name}/`, docsRoot))))).flat().filter(name => name.endsWith('.md'))
    assert.ok(categories.length >= 14)
    assert.ok(files.length >= 60)
  })

  it('requires help metadata and current compliance review', async () => {
    const categories = (await readdir(docsRoot, { withFileTypes: true })).filter(entry => entry.isDirectory())
    for (const category of categories) {
      for (const name of (await readdir(new URL(`${category.name}/`, docsRoot))).filter(file => file.endsWith('.md'))) {
        const raw = await readFile(new URL(`${category.name}/${name}`, docsRoot), 'utf8')
        for (const field of ['category', 'roles', 'segments', 'updated', 'verifiedAgainst', 'relatedTo']) assert.match(raw, new RegExp(`^${field}:`, 'm'), `${category.name}/${name} lacks ${field}`)
        if (category.name === 'compliance') {
          assert.match(raw, /^reviewedBy:/m)
          assert.match(raw, /^reviewedAt:/m)
        }
      }
    }
  })
})
