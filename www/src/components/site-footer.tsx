import { SITE_LINKS } from '../lib/site-links'

const NAV_COLUMNS = [
  {
    heading: 'Product',
    links: [
      { label: 'Features', href: '/#features' },
      { label: 'Pricing', href: '/pricing' },
      { label: 'Documentation', href: '/docs' },
      { label: 'Start studying', href: SITE_LINKS.selfLearner },
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
    <footer style={{ backgroundColor: '#22333b', color: '#b8c6c4' }}>
      <div className="mx-auto max-w-[1200px] px-5 py-14 md:px-10 xl:px-14">
        <div className="grid gap-10 sm:grid-cols-2 lg:grid-cols-[1.4fr_repeat(4,minmax(0,1fr))]">
          <div>
            <div className="flex items-center gap-2.5">
              <img
                src="/logo.svg"
                alt=""
                aria-hidden
                className="h-[34px] w-auto"
              />
              <span className="font-display text-[20px] font-semibold" style={{ color: '#eef4f0' }}>
                Lextures
              </span>
            </div>
            <p className="mt-4 max-w-[24em] text-[14px] leading-[1.6]">
              The learning environment that adapts — quizzing, lessons, grading, and rosters in one
              place.
            </p>
          </div>

          {NAV_COLUMNS.map(({ heading, links }) => (
            <div key={heading}>
              <p
                className="text-[14px] font-bold tracking-[0.03em]"
                style={{ color: '#eef4f0' }}
              >
                {heading}
              </p>
              <ul className="mt-3.5 space-y-2.5">
                {links.map(({ label, href }) => (
                  <li key={label}>
                    <a
                      href={href}
                      className="text-[14px] no-underline transition-colors"
                      style={{ color: '#b8c6c4' }}
                      onMouseEnter={e => {
                        e.currentTarget.style.color = '#eef4f0'
                      }}
                      onMouseLeave={e => {
                        e.currentTarget.style.color = '#b8c6c4'
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
      </div>

      <div style={{ borderTop: '1px solid rgba(255,255,255,0.08)' }}>
        <div
          className="mx-auto flex max-w-[1200px] flex-col gap-2 px-5 py-5 text-[13px] md:px-10 sm:flex-row sm:items-center sm:justify-between xl:px-14"
          style={{ color: '#8ea09d' }}
        >
          <span>© {new Date().getFullYear()} Lextures LLC.</span>
        </div>
      </div>
    </footer>
  )
}
