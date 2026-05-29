import { SITE_LINKS } from '../lib/site-links'

export function SiteFooter() {
  return (
    <footer className="border-t border-stone-200/90 bg-white py-12">
      <div className="mx-auto flex max-w-6xl flex-col gap-10 px-4 sm:flex-row sm:items-start sm:justify-between sm:px-6 lg:px-8">
        <div>
          <div className="flex items-center gap-2.5">
            <img src="/logo.svg" className="h-8 w-8" alt="" aria-hidden />
            <span className="text-base font-semibold text-stone-900">Lextures</span>
          </div>
          <p className="mt-3 max-w-xs text-sm leading-relaxed text-stone-500">
            Open-source LMS for courses, assessments, and institutional workflows. Developed in public
            on GitHub.
          </p>
          <p className="mt-4 text-sm text-stone-400">© {new Date().getFullYear()} Lextures contributors</p>
        </div>
        <div className="flex flex-wrap gap-x-8 gap-y-2 text-sm font-medium text-stone-500">
          <a href={SITE_LINKS.demo} className="no-underline transition-colors hover:text-stone-900">
            Live demo
          </a>
          <a href={SITE_LINKS.github} className="no-underline transition-colors hover:text-stone-900">
            GitHub
          </a>
          <a href="/features" className="no-underline transition-colors hover:text-stone-900">
            Features
          </a>
          <a href="/higher-ed" className="no-underline transition-colors hover:text-stone-900">
            Higher Education
          </a>
          <a href="/k-12" className="no-underline transition-colors hover:text-stone-900">
            K–12
          </a>
          <a href="/self-learner" className="no-underline transition-colors hover:text-stone-900">
            Self-Learner
          </a>
          <a href="/pricing" className="no-underline transition-colors hover:text-stone-900">
            Pricing
          </a>
          <a href="/blog" className="no-underline transition-colors hover:text-stone-900">
            Blog
          </a>
          <a href="/docs" className="no-underline transition-colors hover:text-stone-900">
            Documentation
          </a>
          <a href={SITE_LINKS.privacy} className="no-underline transition-colors hover:text-stone-900">
            Privacy Policy
          </a>
          <a href={SITE_LINKS.terms} className="no-underline transition-colors hover:text-stone-900">
            Terms of Service
          </a>
          <a href={SITE_LINKS.security} className="no-underline transition-colors hover:text-stone-900">
            Security
          </a>
          <a href={SITE_LINKS.accessibility} className="no-underline transition-colors hover:text-stone-900">
            Accessibility
          </a>
          <a href={SITE_LINKS.californiaPrivacyRights} className="no-underline transition-colors hover:text-stone-900">
            California Privacy Rights
          </a>
        </div>
      </div>
    </footer>
  )
}
