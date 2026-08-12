#!/usr/bin/env node
/**
 * @deprecated Use generate-site.mjs (SEO.1). Thin alias for older scripts/docs.
 */
import { spawnSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const generate = path.resolve(path.dirname(fileURLToPath(import.meta.url)), 'generate-site.mjs')
const result = spawnSync(process.execPath, [generate, ...process.argv.slice(2)], {
  stdio: 'inherit',
  env: process.env,
})
process.exit(result.status ?? 1)
