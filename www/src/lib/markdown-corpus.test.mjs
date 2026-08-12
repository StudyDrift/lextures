import assert from 'node:assert/strict'
import { readdir, readFile } from 'node:fs/promises'
import path from 'node:path'
import test from 'node:test'
import { renderMarkdown } from './markdown.ts'
const fixtures = path.resolve(process.cwd(), '../tests/fixtures/content-render')
test('shared marketing content corpus matches the public renderer', async () => { const files=(await readdir(fixtures)).filter(file=>file.endsWith('.md')).sort();assert.ok(files.length>0);for(const file of files){const source=await readFile(path.join(fixtures,file),'utf8');const expected=await readFile(path.join(fixtures,file.replace(/\.md$/,'.expected.html')),'utf8');assert.equal(renderMarkdown(source),expected,file)} })
