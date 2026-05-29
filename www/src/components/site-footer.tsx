import { Github } from 'lucide-react'
import { SITE_LINKS } from '../lib/site-links'

const NAV_COLUMNS = [
  {
    heading: 'Product',
    links: [
      { label: 'Pricing', href: '/pricing' },
      { label: 'Blog', href: '/blog' },
      { label: 'Documentation', href: '/docs' },
      { label: 'Live Demo', href: SITE_LINKS.demo },
    ],
  },
  {
    heading: 'Solutions',
    links: [
      { label: 'Higher Education', href: '/higher-ed' },
      { label: 'K–12', href: '/k-12' },
      { label: 'Self-Learner', href: '/self-learner' },
      { label: 'Get Started', href: '/get-started' },
    ],
  },
  {
    heading: 'Legal',
    links: [
      { label: 'Privacy Policy', href: SITE_LINKS.privacy },
      { label: 'Terms of Service', href: SITE_LINKS.terms },
      { label: 'Security', href: SITE_LINKS.security },
      { label: 'Accessibility', href: SITE_LINKS.accessibility },
      { label: 'California Privacy Rights', href: SITE_LINKS.californiaPrivacyRights },
    ],
  },
]

export function SiteFooter() {
  return (
    <footer className="bg-[#020617] text-slate-400">
      <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
        {/* Main grid */}
        <div className="grid gap-10 border-b border-slate-800 py-14 sm:grid-cols-2 lg:grid-cols-[2fr_1fr_1fr_1fr]">
          {/* Brand column */}
          <div>
            <div className="flex items-center gap-2.5">
              <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-indigo-600 text-white">
                <img src="/logo.svg" className="h-4 w-4 brightness-0 invert" alt="" aria-hidden />
              </span>
              <span className="text-[0.9375rem] font-semibold text-white">Lextures</span>
            </div>
            <p className="mt-4 max-w-xs text-sm leading-relaxed text-slate-500">
              Open-source adaptive LMS for courses, assessments, and institutional workflows.
              Built in public under AGPL-3.0.
            </p>
            <a
              href={SITE_LINKS.github}
              className="mt-5 inline-flex items-center gap-2 text-sm text-slate-500 no-underline transition-colors hover:text-slate-300"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="View Lextures on GitHub"
            >
              <Github className="h-4 w-4" />
              GitHub
            </a>
          </div>

          {/* Nav columns */}
          {NAV_COLUMNS.map(({ heading, links }) => (
            <div key={heading}>
              <p className="text-[0.68rem] font-semibold uppercase tracking-[0.2em] text-slate-500">
                {heading}
              </p>
              <ul className="mt-4 space-y-3">
                {links.map(({ label, href }) => (
                  <li key={label}>
                    <a
                      href={href}
                      className="text-sm text-slate-500 no-underline transition-colors hover:text-slate-200"
                    >
                      {label}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Bottom bar */}
        <div className="flex flex-col items-center justify-between gap-3 py-6 text-xs text-slate-600 sm:flex-row">
          <p>© {new Date().getFullYear()} Lextures contributors. Released under AGPL-3.0.</p>
          <p>Built in public · Self-host for free</p>
        </div>
      </div>
    </footer>
  )
}
