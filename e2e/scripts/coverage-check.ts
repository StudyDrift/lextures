/**
 * E2E.4 — validate completed-feature coverage manifest.
 *
 * Usage: npm run e2e:coverage:check
 * Env:
 *   E2E_COVERAGE_MANIFEST — optional override path
 *   E2E_COVERAGE_DIFF_BASE — optional previous manifest path for CI delta output
 */
import fs from 'node:fs'
import path from 'node:path'
import {
  DEFAULT_MANIFEST_REL,
  REPO_ROOT,
  diffCoverageManifests,
  loadManifest,
  resolveManifestPath,
  validateCompletedFeatureCoverage,
} from '../lib/completed-feature-coverage.js'

function main(): number {
  const started = Date.now()
  const manifestRel = process.env.E2E_COVERAGE_MANIFEST ?? DEFAULT_MANIFEST_REL
  const manifestPath = resolveManifestPath(REPO_ROOT, manifestRel)

  if (!fs.existsSync(manifestPath)) {
    console.error(`coverage-check: manifest not found at ${manifestPath}`)
    return 1
  }

  const errors = validateCompletedFeatureCoverage({
    repoRoot: REPO_ROOT,
    manifestPath,
  })

  const elapsed = Date.now() - started
  if (errors.length > 0) {
    console.error(`coverage-check: FAILED with ${errors.length} error(s) in ${elapsed}ms`)
    for (const err of errors) console.error(`  - ${err}`)
    return 1
  }

  const manifest = loadManifest(manifestPath)
  console.log(
    `coverage-check: OK — ${manifest.entries.length} stories validated in ${elapsed}ms (${path.relative(REPO_ROOT, manifestPath)})`,
  )

  const diffBase = process.env.E2E_COVERAGE_DIFF_BASE
  if (diffBase && fs.existsSync(diffBase)) {
    const before = loadManifest(diffBase)
    const diff = diffCoverageManifests(before, manifest)
    console.log('coverage-check: delta vs base')
    console.log(`  added: ${diff.added.length}`)
    console.log(`  removed: ${diff.removed.length}`)
    console.log(`  levelChanges: ${diff.levelChanges.length}`)
    for (const row of diff.added) console.log(`  + ${row}`)
    for (const row of diff.removed) console.log(`  - ${row}`)
    for (const row of diff.levelChanges) console.log(`  ~ ${row}`)
  }

  return 0
}

process.exit(main())
