/**
 * Minimal client entry for interactive:false routes (SEO.4 FR-4).
 * No React — only nav progressive enhancement, analytics helpers, and web-vitals.
 */
import { initNavEnhancements } from './lib/nav-enhancements'
import { initBlogIndexFilters } from './lib/blog-index-filters'
import { initLiveContent } from './lib/live-content-island'
import { loadAnalytics } from './lib/analytics'
import { initWebVitals } from './lib/web-vitals'

initNavEnhancements()
initBlogIndexFilters()
initLiveContent()
loadAnalytics()
initWebVitals()
