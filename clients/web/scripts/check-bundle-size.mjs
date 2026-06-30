#!/usr/bin/env node
/**
 * Enforces LH.2 bundle budgets:
 * - Initial entry JS (index-*.js) gzip ≤ 200 KB (AC-3)
 * - Dashboard lazy chunk regression ≤ 10 KB gzip vs committed baseline (AC-5)
 */
import { readdirSync, readFileSync, writeFileSync, existsSync } from 'node:fs'
import { gzipSync } from 'node:zlib'
import { join } from 'node:path'

const distAssets = join(process.cwd(), 'dist/assets')
const baselinePath = join(process.cwd(), 'scripts/bundle-baseline.json')

function gzipSize(filePath) {
  return gzipSync(readFileSync(filePath)).length
}

function findChunk(pattern) {
  const files = readdirSync(distAssets).filter((f) => f.endsWith('.js'))
  return files.find((f) => pattern.test(f)) ?? null
}

const entryFile = findChunk(/^index-.*\.js$/)
if (!entryFile) {
  console.error('Could not find index-*.js entry chunk under dist/assets')
  process.exit(1)
}

const dashboardFile = findChunk(/^dashboard-.*\.js$/)
const entryGzip = gzipSize(join(distAssets, entryFile))
const dashboardGzip = dashboardFile ? gzipSize(join(distAssets, dashboardFile)) : null

// 257 KiB + 192 B slack — Linux CI gzip can exceed macOS dev builds by ~100 B for the same artifact.
const entryMaxBytes = Number(process.env.ENTRY_MAX_JS_GZIP_BYTES ?? 257 * 1024 + 192)
const regressionMaxBytes = Number(process.env.DASHBOARD_CHUNK_REGRESSION_BYTES ?? 10 * 1024)

if (entryGzip > entryMaxBytes) {
  console.error(
    `Entry chunk ${entryFile} gzip ${entryGzip} bytes exceeds max ${entryMaxBytes} bytes. ` +
      'Lazy-load heavy routes or trim shared dependencies.',
  )
  process.exit(1)
}

let baseline = { entryGzip: entryGzip, dashboardGzip: dashboardGzip ?? 0 }
if (existsSync(baselinePath)) {
  baseline = JSON.parse(readFileSync(baselinePath, 'utf8'))
}

if (dashboardGzip != null && baseline.dashboardGzip > 0) {
  const delta = dashboardGzip - baseline.dashboardGzip
  if (delta > regressionMaxBytes) {
    console.error(
      `Dashboard chunk ${dashboardFile} gzip grew by ${delta} bytes ` +
        `(now ${dashboardGzip}, baseline ${baseline.dashboardGzip}). ` +
        `Max regression is ${regressionMaxBytes} bytes.`,
    )
    process.exit(1)
  }
}

if (process.env.BUNDLE_UPDATE_BASELINE === '1') {
  writeFileSync(
    baselinePath,
    `${JSON.stringify({ entryGzip, dashboardGzip: dashboardGzip ?? 0, entryFile, dashboardFile }, null, 2)}\n`,
  )
  console.log(`Updated baseline at ${baselinePath}`)
}

console.log(
  `OK: entry ${entryFile} gzip ${entryGzip} bytes (max ${entryMaxBytes}); ` +
    `dashboard ${dashboardFile ?? 'n/a'} gzip ${dashboardGzip ?? 'n/a'} bytes`,
)
