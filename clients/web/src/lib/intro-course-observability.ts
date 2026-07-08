/** Lightweight client-only intro course funnel metrics (IC06; no PII). */

const CARD_VIEW_KEY = 'intro_course.card_view'
const CTA_CLICK_KEY = 'intro_course.cta_click'
const BANNER_DISMISS_KEY = 'intro_course.banner_dismiss'
const CELEBRATION_VIEW_KEY = 'intro_course.completed_celebration_view'

function bumpCounter(key: string): void {
  try {
    const prev = Number.parseInt(sessionStorage.getItem(key) ?? '0', 10)
    sessionStorage.setItem(key, String(Number.isFinite(prev) ? prev + 1 : 1))
  } catch {
    // ignore storage errors
  }
}

export function recordIntroCourseCardView(): void {
  bumpCounter(CARD_VIEW_KEY)
}

export function recordIntroCourseCtaClick(): void {
  bumpCounter(CTA_CLICK_KEY)
}

export function recordIntroCourseBannerDismiss(): void {
  bumpCounter(BANNER_DISMISS_KEY)
}

export function recordIntroCourseCelebrationView(): void {
  bumpCounter(CELEBRATION_VIEW_KEY)
}