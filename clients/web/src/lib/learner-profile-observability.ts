/** Lightweight client-only interaction metrics (LP07 observability; no PII). */

const PAGE_VIEW_KEY = 'learner_profile.page_view'
const EVIDENCE_EXPAND_KEY = 'learner_profile.evidence_expanded'

function bumpCounter(key: string): void {
  try {
    const prev = Number.parseInt(sessionStorage.getItem(key) ?? '0', 10)
    sessionStorage.setItem(key, String(Number.isFinite(prev) ? prev + 1 : 1))
  } catch {
    // ignore storage errors
  }
}

export function recordLearnerProfilePageView(): void {
  bumpCounter(PAGE_VIEW_KEY)
}

export function recordLearnerProfileEvidenceExpanded(facetKey: string, insightKey: string): void {
  bumpCounter(EVIDENCE_EXPAND_KEY)
  try {
    const detailKey = `${EVIDENCE_EXPAND_KEY}.${facetKey}.${insightKey}`
    bumpCounter(detailKey)
  } catch {
    // ignore
  }
}