import { StrictMode } from 'react'
import { hydrateRoot, createRoot } from 'react-dom/client'
import './index.css'
import App from './app.tsx'
import { SsrDataProvider } from './lib/ssr-context'
import { readClientSsrData } from './lib/ssr-data'
import { isInteractiveRoute, loadRouteElement } from './lib/route-manifest'
import { loadAnalytics } from './lib/analytics'
import { initWebVitals } from './lib/web-vitals'
import { initNavEnhancements } from './lib/nav-enhancements'

const rootEl = document.getElementById('root')
if (!rootEl) {
  throw new Error('Missing #root element')
}

const path =
  typeof window !== 'undefined'
    ? window.location.pathname.replace(/\/+$/, '') || '/'
    : '/'

// SEO.4 FR-7 / FR-16 / FR-17 — analytics + vitals after idle, never on critical path.
loadAnalytics()
initWebVitals()

const interactiveAttr = document.documentElement.dataset.interactive
const interactive =
  interactiveAttr === 'true'
    ? true
    : interactiveAttr === 'false'
      ? false
      : isInteractiveRoute(path)

// Progressive enhancement for header/menu on all pages (works without React).
initNavEnhancements()

if (!interactive && rootEl.hasChildNodes()) {
  // SEO.4 FR-4 — content pages: prerendered HTML only, no React hydration.
  // Keep SSR markup as-is; nav-enhancements handles mobile menu.
  if (import.meta.env.DEV) {
    // eslint-disable-next-line no-console
    console.debug('[www] interactive:false — skip React hydration for', path)
  }
} else {
  // Vite dev serves the shared, empty index.html for every pathname. Static
  // routes therefore still need a client render when no prerendered markup is
  // present; generated production pages keep taking the no-hydration branch.
  const ssrData = readClientSsrData()

  void loadRouteElement(path).then(ssrPage => {
    const app = (
      <StrictMode>
        <SsrDataProvider data={ssrData}>
          <App ssrPage={ssrPage} />
        </SsrDataProvider>
      </StrictMode>
    )

    if (rootEl.hasChildNodes()) {
      hydrateRoot(rootEl, app)
    } else {
      createRoot(rootEl).render(app)
    }
  })
}
