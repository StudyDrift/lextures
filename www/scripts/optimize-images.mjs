#!/usr/bin/env node
/**
 * SEO.4 FR-12 / FR-13 — convert public raster images to AVIF + WebP alongside PNG.
 * Runs as part of `npm run build` before generate-site (idempotent).
 *
 * Keeps original PNG for fallback; emits .avif and .webp next to each source.
 */

import { readdir, stat } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import sharp from 'sharp'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..')
const PUBLIC = path.join(ROOT, 'public')

const ROOTS = [
  path.join(PUBLIC, 'assets', 'screenshots'),
  PUBLIC, // docs-*.png at public root
]

async function listPngs(dir) {
  /** @type {string[]} */
  const out = []
  let entries
  try {
    entries = await readdir(dir, { withFileTypes: true })
  } catch {
    return out
  }
  for (const ent of entries) {
    if (!ent.isFile()) continue
    if (!/\.png$/i.test(ent.name)) continue
    // Only docs-* at public root
    if (dir === PUBLIC && !ent.name.startsWith('docs-')) continue
    out.push(path.join(dir, ent.name))
  }
  return out
}

async function convertOne(src) {
  const base = src.replace(/\.png$/i, '')
  const avifPath = `${base}.avif`
  const webpPath = `${base}.webp`
  const img = sharp(src).rotate()
  const meta = await img.metadata()
  const width = meta.width && meta.width > 1600 ? 1600 : meta.width

  const pipeline = width ? img.resize({ width, withoutEnlargement: true }) : img

  await Promise.all([
    pipeline
      .clone()
      .avif({ quality: 55, effort: 4 })
      .toFile(avifPath),
    pipeline
      .clone()
      .webp({ quality: 72 })
      .toFile(webpPath),
  ])

  const [srcSt, avifSt, webpSt] = await Promise.all([stat(src), stat(avifPath), stat(webpPath)])
  return {
    src: path.relative(PUBLIC, src),
    png: srcSt.size,
    avif: avifSt.size,
    webp: webpSt.size,
    width: meta.width,
    height: meta.height,
  }
}

export async function optimizeImages() {
  const files = []
  for (const root of ROOTS) {
    files.push(...(await listPngs(root)))
  }
  if (!files.length) {
    console.log('[optimize-images] no PNGs found')
    return []
  }

  const results = []
  for (const f of files) {
    try {
      results.push(await convertOne(f))
    } catch (err) {
      console.warn(`[optimize-images] skip ${f}: ${err.message || err}`)
    }
  }

  const pngTotal = results.reduce((s, r) => s + r.png, 0)
  const avifTotal = results.reduce((s, r) => s + r.avif, 0)
  console.log(
    `[optimize-images] ${results.length} image(s) · PNG ${Math.round(pngTotal / 1024)} KB → AVIF ${Math.round(avifTotal / 1024)} KB`,
  )
  return results
}

if (import.meta.url === `file://${process.argv[1]}` || process.argv[1]?.endsWith('optimize-images.mjs')) {
  optimizeImages().catch(err => {
    console.error(err)
    process.exit(1)
  })
}
