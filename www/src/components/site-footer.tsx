import { SITE_LINKS } from '../lib/site-links'

const NAV_COLUMNS = [
  {
    heading: 'Product',
    links: [
      { label: 'Features', href: '/#features' },
      { label: 'Pricing', href: '/pricing' },
      { label: 'Documentation', href: '/docs' },
      { label: 'Live demo', href: SITE_LINKS.demo },
    ],
  },
  {
    heading: 'Institutions',
    links: [
      { label: 'Higher education', href: '/higher-ed' },
      { label: 'K–12', href: '/k-12' },
      { label: 'Parents', href: '/parents' },
      { label: 'Self-learners', href: '/self-learner' },
    ],
  },
  {
    heading: 'Project',
    links: [
      { label: 'GitHub', href: SITE_LINKS.github },
      { label: 'Blog', href: '/blog' },
      { label: 'Self-hosting', href: '/docs/self-hosting' },
      { label: 'Security', href: SITE_LINKS.security },
    ],
  },
  {
    heading: 'Legal',
    links: [
      { label: 'Privacy policy', href: SITE_LINKS.privacy },
      { label: 'Terms of service', href: SITE_LINKS.terms },
      { label: 'Accessibility', href: SITE_LINKS.accessibility },
      { label: 'California privacy rights', href: SITE_LINKS.californiaPrivacyRights },
    ],
  },
]

export function SiteFooter() {
  return (
    <footer className="border-t" style={{ backgroundColor: 'var(--paper)', borderColor: 'var(--line)' }}>
      <div className="mx-auto max-w-[1200px] px-5 py-14 md:px-10 xl:px-14">
        <div className="grid gap-10 sm:grid-cols-2 lg:grid-cols-[1.4fr_repeat(4,minmax(0,1fr))]">
          <div>
            <div className="flex items-center gap-3">
              <img
                src="/assets/lextures-mark.svg"
                alt=""
                aria-hidden
                className="h-7 w-7"
                width={28}
                height={28}
              />
              <span
                className="font-display text-[20px] font-semibold"
                style={{ color: 'var(--ink-nav)' }}
              >
                Lextures
              </span>
            </div>
            <p className="mt-4 max-w-xs text-[15px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
              Open-source LMS for courses, assessments, and the workflows that surround them.
            </p>
          </div>

          {NAV_COLUMNS.map(({ heading, links }) => (
            <div key={heading}>
              <p className="section-label">{heading}</p>
              <ul className="mt-4 space-y-2.5">
                {links.map(({ label, href }) => (
                  <li key={label}>
                    <a
                      href={href}
                      className="text-[14px] no-underline transition-colors"
                      style={{ color: 'var(--text-soft)' }}
                      onMouseEnter={e => {
                        e.currentTarget.style.color = 'var(--ink-nav)'
                      }}
                      onMouseLeave={e => {
                        e.currentTarget.style.color = 'var(--text-soft)'
                      }}
                    >
                      {label}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div
          className="mt-12 flex flex-col gap-2 border-t pt-6 text-[13px] sm:flex-row sm:items-center sm:justify-between"
          style={{ borderColor: 'var(--line)', color: 'var(--muted)' }}
        >
          <p>© {new Date().getFullYear()} Lextures contributors. Released under AGPL-3.0.</p>
          <p>Self-host on Postgres · LTI 1.3 · SCIM 2.0</p>
        </div>
      </div>
    </footer>
  )
}
