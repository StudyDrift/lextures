import assert from 'node:assert/strict'
import { mkdtemp, readFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { describe, it } from 'node:test'
import sharp from 'sharp'
import { cardHash, cardSvg, renderOgCard } from './render.mjs'

describe('SEO.14 social cards', () => {
  it('is deterministic and escapes hostile long titles', () => {
    const input = { title: '<script>alert(1)</script> 📚 ' + 'long '.repeat(40), section: 'Guide' }
    assert.equal(cardHash(input), cardHash(input))
    const svg = cardSvg(input)
    assert.doesNotMatch(svg, /<script>/)
    assert.match(svg, /…/)
  })
  it('renders a compact 1200 by 630 raster card', async () => {
    const dir = await mkdtemp(path.join(os.tmpdir(), 'lextures-og-'))
    const card = await renderOgCard({ title: 'Adaptive assessment explained', routePath: '/blog/adaptive', distDir: dir })
    const bytes = await readFile(path.join(dir, card.relativePath))
    const metadata = await sharp(bytes).metadata()
    assert.equal(metadata.format, 'png'); assert.equal(metadata.width, 1200); assert.equal(metadata.height, 630)
    assert.ok(bytes.length <= 300 * 1024)
  })
})
