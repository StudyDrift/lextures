#!/usr/bin/env node
// Optional regeneration: the site build reuses checked-in images when Playwright or a demo URL is absent.
import { readdir, readFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const demoUrl = process.env.HELP_SCREENSHOT_BASE_URL
if (!demoUrl) {
  console.warn('WARN: HELP_SCREENSHOT_BASE_URL is unset; reusing existing help screenshots.')
  process.exit(0)
}
const seed = await readFile(path.join(root, 'scripts/screenshots/synthetic-seed.json'), 'utf8')
if (!JSON.parse(seed).syntheticOnly) throw new Error('Screenshot seed must assert syntheticOnly.')
const { chromium } = await import('playwright')
const browser = await chromium.launch()
try {
  const page = await browser.newPage({ locale: process.env.HELP_SCREENSHOT_LOCALE || 'en-US' })
  await page.goto(demoUrl, { waitUntil: 'networkidle' })
  // Capture specs are added beside an article when a UI image is necessary. Text remains authoritative.
  const specs = (await readdir(path.join(root, 'scripts/screenshots'))).filter(name => name.endsWith('.spec.json'))
  console.log(`Validated synthetic demo tenant; ${specs.length} screenshot specs available.`)
} finally { await browser.close() }
