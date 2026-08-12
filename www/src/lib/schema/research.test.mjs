import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

test('research schema declares required dataset properties', () => {
  const source = readFileSync(new URL('./research.ts', import.meta.url), 'utf8')
  for (const field of ["'@type': 'Dataset'", 'license:', 'distribution:', 'measurementTechnique:', 'variableMeasured:', 'creator:']) assert.match(source, new RegExp(field.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
})

