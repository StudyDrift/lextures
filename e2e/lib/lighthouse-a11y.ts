/**
 * Lighthouse accessibility helpers (LH.3).
 * Parses committed or fresh Lighthouse JSON and validates accessibility thresholds.
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import type { Result } from 'lighthouse'

import { REPO_ROOT } from './lighthouse-harness.js'

export const DEFAULT_MIN_ACCESSIBILITY_SCORE = 0.95

export const WEIGHTED_A11Y_AUDIT_IDS = [
  'button-name',
  'color-contrast',
  'heading-order',
  'link-name',
  'image-alt',
] as const

export type WeightedA11yAuditId = (typeof WEIGHTED_A11Y_AUDIT_IDS)[number]

export interface A11yAuditFailure {
  id: string
  title: string
  weight: number
  score: number | null
  failingElements: string[]
}

export interface AccessibilityReportSummary {
  score: number
  failures: A11yAuditFailure[]
  failureCount: number
}

type LighthouseReport = Pick<Result, 'categories' | 'audits'>

export function parseLighthouseReportJson(raw: string): LighthouseReport {
  return JSON.parse(raw) as LighthouseReport
}

export function loadCommittedDashboardReport(): LighthouseReport {
  const path = join(REPO_ROOT, 'docs/lighthouse/global-dashboard-darkmode.json')
  return parseLighthouseReportJson(readFileSync(path, 'utf8'))
}

/** Collect weighted accessibility audits that did not pass. */
export function summarizeAccessibilityFailures(
  report: LighthouseReport,
): AccessibilityReportSummary {
  const score = report.categories.accessibility?.score
  if (typeof score !== 'number') {
    throw new Error('categories.accessibility.score is missing or not a number')
  }

  const auditRefs = report.categories.accessibility?.auditRefs ?? []
  const failures: A11yAuditFailure[] = []

  for (const ref of auditRefs) {
    if (ref.weight <= 0) continue
    const audit = report.audits[ref.id]
    if (!audit) continue
    if (audit.score === null || audit.score >= 1) continue

    const items = audit.details && 'items' in audit.details ? audit.details.items : []
    const failingElements = Array.isArray(items)
      ? items
          .slice(0, 5)
          .map((item) => {
            if (item && typeof item === 'object' && 'node' in item) {
              const node = item.node as { selector?: string; snippet?: string }
              return node.selector ?? node.snippet ?? '(unknown element)'
            }
            return '(unknown element)'
          })
      : []

    failures.push({
      id: ref.id,
      title: audit.title,
      weight: ref.weight,
      score: audit.score,
      failingElements,
    })
  }

  return {
    score,
    failures,
    failureCount: failures.length,
  }
}

export function assertAccessibilityScore(
  report: LighthouseReport,
  minScore = DEFAULT_MIN_ACCESSIBILITY_SCORE,
): AccessibilityReportSummary {
  const summary = summarizeAccessibilityFailures(report)

  if (summary.score < minScore) {
    const details = summary.failures
      .map(
        (f) =>
          `  - ${f.id} (weight ${f.weight}): ${f.failingElements.join(', ') || 'no selector'}`,
      )
      .join('\n')
    throw new Error(
      `Lighthouse accessibility score ${summary.score.toFixed(2)} is below minimum ${minScore}.\n` +
        `Failing weighted audits (${summary.failureCount}):\n${details || '  (none listed)'}`,
    )
  }

  return summary
}

export function resolveMinAccessibilityScore(): number {
  const raw = process.env.LH_MIN_A11Y_SCORE?.trim()
  if (!raw) return DEFAULT_MIN_ACCESSIBILITY_SCORE
  const parsed = Number.parseFloat(raw)
  if (!Number.isFinite(parsed) || parsed < 0 || parsed > 1) {
    throw new Error(`Invalid LH_MIN_A11Y_SCORE: ${raw} (expected 0–1)`)
  }
  return parsed
}
