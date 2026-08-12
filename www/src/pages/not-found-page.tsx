import { MarketingPageShell } from '../components/marketing-page-shell'

const HUB_LINKS = [
  { href: '/', label: 'Home' },
  { href: '/docs', label: 'Docs' },
  { href: '/blog', label: 'Blog' },
  { href: '/pricing', label: 'Pricing' },
  { href: '/courses', label: 'Courses' },
] as const

export function NotFoundPage() {
  return (
    <MarketingPageShell>
      <div className="mx-auto max-w-[640px] px-5 py-24 text-center md:px-10">
        <p
          className="text-[13px] font-semibold uppercase tracking-[0.06em]"
          style={{ color: 'var(--coral)' }}
        >
          404
        </p>
        <h1
          className="font-display mt-4 text-[clamp(28px,4vw,40px)] font-semibold leading-tight"
          style={{ color: 'var(--ink-nav)' }}
        >
          Page not found
        </h1>
        <p className="mt-4 text-[17px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
          That URL is not part of the Lextures site. Try one of the hubs below, or start from the
          homepage.
        </p>
        <nav aria-label="Suggested pages" className="mt-10">
          <ul className="flex flex-wrap items-center justify-center gap-3">
            {HUB_LINKS.map(link => (
              <li key={link.href}>
                <a href={link.href} className="btn-secondary">
                  {link.label}
                </a>
              </li>
            ))}
          </ul>
        </nav>
      </div>
    </MarketingPageShell>
  )
}
