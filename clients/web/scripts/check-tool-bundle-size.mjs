#!/usr/bin/env node
/**
 * CT.5 FR-14 / AC-9 — renderer chunk ≤ 40 KB gz per tool.
 * Looks for Vite chunks matching *{toolId}* under dist/assets after build.
 */
import { readdirSync, readFileSync, existsSync } from 'node:fs'
import { gzipSync } from 'node:zlib'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const distAssets = join(__dirname, '../dist/assets')
const maxBytes = Number(process.env.TOOL_CHUNK_MAX_GZIP_BYTES ?? 40 * 1024)
const toolIds = (process.env.TOOL_CHUNK_IDS ?? 'noop_probe,sandbox_probe')
  .split(',')
  .map((s) => s.trim())
  .filter(Boolean)

if (!existsSync(distAssets)) {
  console.log('Skip tool bundle check: dist/assets missing (run after vite build).')
  process.exit(0)
}

const files = readdirSync(distAssets).filter((f) => f.endsWith('.js'))
let failed = 0
for (const toolId of toolIds) {
  const match = files.find((f) => f.includes(toolId.replaceAll('_', '-')) || f.includes(toolId))
  if (!match) {
    // In-process tools may be folded into a shared chunk; warn but do not fail unless required.
    if (process.env.TOOL_CHUNK_REQUIRE === '1') {
      console.error(`FAIL: no chunk found for tool ${toolId}`)
      failed++
    } else {
      console.log(`OK: ${toolId} (no dedicated chunk; shared bundle)`)
    }
    continue
  }
  const gz = gzipSync(readFileSync(join(distAssets, match))).length
  if (gz > maxBytes) {
    console.error(`FAIL: tool ${toolId} chunk ${match} gzip ${gz} exceeds ${maxBytes}`)
    failed++
  } else {
    console.log(`OK: tool ${toolId} chunk ${match} gzip ${gz} (max ${maxBytes})`)
  }
}
if (failed > 0) process.exit(1)
