#!/usr/bin/env npx tsx
/**
 * LH.1 — Reproducible Lighthouse harness for the signed-in global dashboard.
 *
 * Prerequisites: API + web client running (e.g. `make dev` or e2e-local stack).
 *
 * Usage:
 *   npm run lighthouse:dashboard:dark          # from e2e/ or clients/web/
 *   THEME=light npm run lighthouse:dashboard:dark
 *   LH_REQUIRE_AUTH=1 npm run lighthouse:dashboard:dark   # fail without LH_TOKEN (AC-3)
 */
import { parseHarnessEnv, runLighthouseDashboard } from '../lib/lighthouse-harness.js'

async function main() {
  const options = parseHarnessEnv()
  const result = await runLighthouseDashboard(options)

  const perfPct = Math.round(result.performanceScore * 100)
  const a11yPct = Math.round(result.accessibilityScore * 100)
  const { failureCount } = result.accessibilitySummary

  console.log(`Lighthouse report written to ${result.outputPath}`)
  console.log(`  requestedUrl: ${result.requestedUrl}`)
  console.log(`  performance: ${perfPct}`)
  console.log(`  accessibility: ${a11yPct} (${failureCount} weighted audit failure(s))`)
}

main().catch((err: unknown) => {
  const message = err instanceof Error ? err.message : String(err)
  console.error(`Lighthouse harness failed: ${message}`)
  process.exit(1)
})
