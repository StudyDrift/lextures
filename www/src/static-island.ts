/**
 * Minimal client entry for interactive:false routes (SEO.4 FR-4).
 * No React — only nav progressive enhancement, deferred analytics, and web-vitals.
 */
import { initNavEnhancements } from './lib/nav-enhancements'
import { loadAnalytics } from './lib/analytics'
import { initWebVitals } from './lib/web-vitals'

initNavEnhancements()
loadAnalytics()
initWebVitals()
