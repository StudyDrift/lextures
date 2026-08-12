import { useEffect } from 'react'
import { Header } from '../components/header'
import { SiteFooter } from '../components/site-footer'
import { CtaSection } from '../components/home/cta-section'
import { FeatureHighlights } from '../components/home/feature-highlights'
import { HeroSection } from '../components/home/hero-section'
import { QuoteSection } from '../components/home/quote-section'
import { WorkflowSection } from '../components/home/workflow-section'

export function HomePage() {
  useEffect(() => {
    if (typeof window === 'undefined') return
    const hash = window.location.hash
    if (hash && !hash.startsWith('#/')) {
      document.querySelector(hash)?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [])

  return (
    <div
      className="min-h-screen overflow-x-hidden antialiased"
      style={{ backgroundColor: 'var(--paper)', color: 'var(--text)' }}
    >
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
        <QuoteSection />
        <CtaSection />
      </main>

      <SiteFooter />
    </div>
  )
}
