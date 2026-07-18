/**
 * E2E.4 — generate Markdown coverage report (artifact; do not hand-edit).
 *
 * Usage: npm run e2e:coverage:report
 */
import fs from 'node:fs'
import path from 'node:path'
import {
  DEFAULT_MANIFEST_REL,
  DEFAULT_REPORT_REL,
  REPO_ROOT,
  loadManifest,
  renderCoverageReportMarkdown,
  resolveManifestPath,
  summarizeCoverage,
  validateCompletedFeatureCoverage,
} from '../lib/completed-feature-coverage.js'

function main(): number {
  const manifestRel = process.env.E2E_COVERAGE_MANIFEST ?? DEFAULT_MANIFEST_REL
  const reportRel = process.env.E2E_COVERAGE_REPORT ?? DEFAULT_REPORT_REL
  const manifestPath = resolveManifestPath(REPO_ROOT, manifestRel)
  const reportPath = resolveManifestPath(REPO_ROOT, reportRel)

  const errors = validateCompletedFeatureCoverage({
    repoRoot: REPO_ROOT,
    manifestPath,
  })
  if (errors.length > 0) {
    console.error(`coverage-report: refusing to write report; ${errors.length} validation error(s)`)
    for (const err of errors.slice(0, 40)) console.error(`  - ${err}`)
    if (errors.length > 40) console.error(`  … ${errors.length - 40} more`)
    return 1
  }

  const manifest = loadManifest(manifestPath)
  const summary = summarizeCoverage(manifest)
  const md = renderCoverageReportMarkdown(summary)
  fs.mkdirSync(path.dirname(reportPath), { recursive: true })
  fs.writeFileSync(reportPath, md, 'utf8')
  console.log(
    `coverage-report: wrote ${path.relative(REPO_ROOT, reportPath)} (${summary.totalStories} stories; missing=${summary.byCoverage.missing})`,
  )
  return 0
}

process.exit(main())
