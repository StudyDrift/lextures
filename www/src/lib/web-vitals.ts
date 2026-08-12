/**
 * Field Core Web Vitals collector (SEO.4 FR-17).
 * Lazy-loads web-vitals (≤ ~2 KB gzip) and reports to GA4 via gtag when present.
 * Never blocks interaction handlers (FR-16).
 */

export type WebVitalMetric = {
  name: string
  value: number
  rating: string
  id: string
  navigationType?: string
  delta?: number
}

type GtagFn = (...args: unknown[]) => void

function getGtag(): GtagFn | null {
  if (typeof window === 'undefined') return null
  const w = window as Window & { gtag?: GtagFn; dataLayer?: unknown[] }
  if (typeof w.gtag === 'function') return w.gtag
  return null
}

function reportMetric(metric: WebVitalMetric, elementSelector?: string): void {
  const gtag = getGtag()
  const payload = {
    event_category: 'Web Vitals',
    event_label: metric.id,
    value: Math.round(metric.name === 'CLS' ? metric.value * 1000 : metric.value),
    metric_name: metric.name,
    metric_value: metric.value,
    metric_rating: metric.rating,
    metric_delta: metric.delta,
    page_path: typeof location !== 'undefined' ? location.pathname : '',
    navigation_type: metric.navigationType || '',
    element_selector: elementSelector || '',
    non_interaction: true,
  }

  if (gtag) {
    gtag('event', 'web_vitals', payload)
    return
  }

  // Dev / no-GA fallback for local debugging.
  if (import.meta.env.DEV) {
    // eslint-disable-next-line no-console
    console.debug('[web-vitals]', metric.name, metric.value, metric.rating, elementSelector)
  }
}

/**
 * Start collecting LCP, INP, CLS, TTFB (and FCP when available).
 * Call once after idle on interactive pages; safe on static pages too.
 */
export function initWebVitals(): void {
  if (typeof window === 'undefined') return

  const start = () => {
    void import('web-vitals/attribution').then(({ onLCP, onINP, onCLS, onTTFB, onFCP }) => {
      onLCP(m => {
        const el = m.attribution?.target
        reportMetric(m, el || '')
      })
      onINP(m => {
        const target = m.attribution?.interactionTarget
        reportMetric(m, target || '')
      })
      onCLS(m => reportMetric(m))
      onTTFB(m => reportMetric(m))
      onFCP(m => reportMetric(m))
    })
  }

  const ric = (
    window as Window & {
      requestIdleCallback?: (cb: () => void, opts?: { timeout: number }) => number
    }
  ).requestIdleCallback

  if (typeof ric === 'function') {
    ric(() => start(), { timeout: 4000 })
  } else {
    setTimeout(start, 1)
  }
}
