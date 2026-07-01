import { useEffect, useState } from 'react'
import { Header } from './components/header'
import { SiteFooter } from './components/site-footer'
import { DocsCtaSection } from './components/home/docs-cta-section'
import { FeatureHighlights } from './components/home/feature-highlights'
import { HeroSection } from './components/home/hero-section'
import { VibeActivitiesSection } from './components/home/vibe-activities-section'
import { WhoItsForSection } from './components/home/who-its-for-section'
import { WorkflowSection } from './components/home/workflow-section'
import { BlogIndex } from './pages/blog-index'
import { BlogPost } from './pages/blog-post'
import { DocsIndex } from './pages/docs-index'
import { DocsPost } from './pages/docs-post'
import { ParentsPage } from './pages/parents-page'
import { GetStartedPage } from './pages/get-started-page'
import { HigherEdPage } from './pages/higher-ed-page'
import { K12Page } from './pages/k12-page'
import { PricingPage } from './pages/pricing-page'
import { RequestInformationPage } from './pages/request-information-page'
import { SelfLearnerPage } from './pages/self-learner-page'
import {
  PrivacyPolicyHistoryPage,
  PrivacyPolicyPage,
  TermsOfServiceHistoryPage,
  TermsOfServicePage,
} from './pages/legal-pages'
import { SecurityPage } from './pages/security-page'
import { AccessibilityConformancePage } from './pages/accessibility-conformance-page'
import { CaliforniaPrivacyRightsPage } from './pages/california-privacy-rights-page'
import { VpatPage } from './pages/vpat-page'

function useHashRoute() {
  const [hash, setHash] = useState(() => window.location.hash)
  useEffect(() => {
    const handler = () => setHash(window.location.hash)
    window.addEventListener('hashchange', handler)
    return () => window.removeEventListener('hashchange', handler)
  }, [])
  return hash
}

/* ─────────────────────────────────────────────────────────
   HOMEPAGE
   ───────────────────────────────────────────────────────── */
function HomePage() {
  useEffect(() => {
    const hash = window.location.hash
    if (hash && !hash.startsWith('#/')) {
      document.querySelector(hash)?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [])

  return (
    <div className="min-h-screen antialiased" style={{ backgroundColor: 'var(--paper)', color: 'var(--text)' }}>
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[100] focus:rounded focus:px-4 focus:py-2 focus:text-sm"
        style={{ backgroundColor: 'var(--ink-nav)', color: 'var(--cream)' }}
      >
        Skip to content
      </a>

      <Header />

      <main id="main">
        <HeroSection />
        <FeatureHighlights />
        <WorkflowSection />
        <VibeActivitiesSection />
        <WhoItsForSection />
        <DocsCtaSection />
      </main>

      <SiteFooter />
    </div>
  )
}

/* ─────────────────────────────────────────────────────────
   ROUTER
   ───────────────────────────────────────────────────────── */
function resolveRoute(pathname: string, hash: string): string {
  if (hash.startsWith('#/')) {
    const hashRoute = hash.slice(1)
    if (pathname !== '/' && hashRoute.startsWith(`${pathname}/`)) {
      return hashRoute
    }
    if (pathname === '/') return hashRoute
  }
  return pathname !== '/' ? pathname : '/'
}

export default function App() {
  const hash = useHashRoute()
  const route = resolveRoute(window.location.pathname, hash)

  if (route === '/get-started') return <GetStartedPage />
  if (route === '/parents') return <ParentsPage />
  if (route === '/higher-ed') return <HigherEdPage />
  if (route === '/k-12') return <K12Page />
  if (route === '/self-learner') return <SelfLearnerPage />
  if (route === '/pricing') return <PricingPage />
  if (route === '/request-information') return <RequestInformationPage />
  if (route === '/blog') return <BlogIndex />
  if (route.startsWith('/blog/')) return <BlogPost slug={route.slice('/blog/'.length)} />
  if (route === '/docs') return <DocsIndex />
  if (route.startsWith('/docs/')) return <DocsPost slug={route.slice('/docs/'.length)} />
  if (route === '/privacy') return <PrivacyPolicyPage />
  if (route === '/privacy/history') return <PrivacyPolicyHistoryPage />
  if (route === '/terms') return <TermsOfServicePage />
  if (route === '/terms/history') return <TermsOfServiceHistoryPage />
  if (route === '/security') return <SecurityPage />
  if (route === '/accessibility') return <AccessibilityConformancePage />
  if (route === '/accessibility/vpat') return <VpatPage />
  if (route === '/privacy-rights/california') return <CaliforniaPrivacyRightsPage />
  return <HomePage />
}
