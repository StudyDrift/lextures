import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { renderMarkdown } from '../src/lib/markdown.ts'

const wwwRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const corpus = path.resolve(wwwRoot, '../tests/fixtures/content-render')
await mkdir(corpus, { recursive: true })

const fixtures = [
  ['002-all-directives', `:::definition term="Rubric"\nA scoring guide.\n:::\n\n:::comparison-table summary="Options"\n| A | B |\n|---|---|\n| 1 | 2 |\n:::\n\n:::steps\n1. Start.\n2. Finish.\n:::\n\n:::faq\n### What is it?\nAn answer.\n### When is it used?\nDuring work.\n### Why does it help?\nIt clarifies expectations.\n:::\n\n:::stat 42 percent in 2026\n42%\n:::\n\n:::sources\n[^1]: https://example.com\n:::\n`],
  ['003-malformed-and-unknown', `:::wat\n<script>alert(1)</script>\n\n[x](javascript:alert(1))\n`],
]

const blogs = [
  'adaptive-ai-and-education', 'blooms-taxonomy-in-the-age-of-ai',
  'effective-rubrics-in-the-age-of-ai', 'rethinking-assessment-in-the-ai-era',
  'the-synthetic-renaissance',
]
for (const [name, source] of fixtures) await writeFile(path.join(corpus, `${name}.md`), source)
for (const [index, name] of blogs.entries()) {
  const raw = await readFile(path.join(wwwRoot, 'src/blog', `${name}.md`), 'utf8')
  const source = raw.replace(/^---\r?\n[\s\S]*?\r?\n---\r?\n/, '')
  fixtures.push([`${String(index + 4).padStart(3, '0')}-${name}`, source])
  await writeFile(path.join(corpus, `${String(index + 4).padStart(3, '0')}-${name}.md`), source)
}
for (const [name, source] of fixtures) await writeFile(path.join(corpus, `${name}.expected.html`), renderMarkdown(source))
