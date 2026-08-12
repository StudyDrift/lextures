import { ChevronDown, Menu, X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { Breadcrumbs } from './breadcrumbs'
import { LocaleSwitcher } from './locale-controls'

const NAV_LINKS = [
  { label: 'Pricing', href: '/pricing' },
  { label: 'Docs', href: '/docs' },
] as const

const AUDIENCE_LINKS = [
  { label: 'Higher education', href: '/higher-ed' },
  { label: 'K–12', href: '/k12' },
  { label: 'Parents', href: '/parents' },
  { label: 'Homeschool', href: '/homeschool' },
] as const

const PLATFORM_LINKS = [
  { label: 'Platform overview', href: '/platform' },
  { label: 'Adaptive learning', href: '/platform/adaptive-learning' },
  { label: 'Assessment', href: '/platform/assessment' },
  { label: 'Grading', href: '/platform/grading' },
  { label: 'Analytics', href: '/platform/analytics' },
  { label: 'Accessibility', href: '/platform/accessibility' },
  { label: 'AI', href: '/platform/ai' },
] as const

const RESOURCE_LINKS = [
  { label: 'Resources overview', href: '/resources' }, { label: 'Blog', href: '/blog' },
  { label: 'Original research', href: '/resources/research' },
  { label: 'Guides', href: '/guides' }, { label: 'Glossary', href: '/glossary' },
  { label: 'Research', href: '/research' }, { label: 'Templates', href: '/templates' },
  { label: 'Standards', href: '/standards' }, { label: 'Tools', href: '/tools' },
] as const

function Logo() {
  return (
    <a href="/" className="flex items-center gap-3 no-underline" aria-label="Lextures home">
      <img
        src="/assets/lextures-mark.svg"
        alt=""
        aria-hidden
        className="h-8 w-8"
        width={32}
        height={32}
      />
      <span
        className="font-display text-[23px] font-semibold leading-none"
        style={{ color: 'var(--ink-nav)' }}
      >
        Lextures
      </span>
    </a>
  )
}

/**
 * Audience dropdown — React state when hydrated; data-* attrs allow the
 * static-island script to drive the same markup without React (SEO.4 FR-4).
 */
function NavDropdown({ label, links, onNavigate }: { label: string; links: ReadonlyArray<{label: string; href: string}>; onNavigate?: () => void }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen(v => !v)}
        aria-expanded={open}
        aria-haspopup="true"
        data-nav-dropdown
        onKeyDown={event => {
          if (event.key === 'Escape') { setOpen(false); event.currentTarget.focus() }
          if (event.key === 'ArrowDown') {
            event.preventDefault(); setOpen(true)
            requestAnimationFrame(() => ref.current?.querySelector<HTMLAnchorElement>('a')?.focus())
          }
        }}
        className="flex cursor-pointer items-center gap-1 text-[15px] font-medium"
        style={{ color: 'var(--text)' }}
      >
        {label}
        <ChevronDown
          className={`h-3.5 w-3.5 transition-transform duration-150 ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      <div
        data-nav-dropdown-panel
        hidden={!open}
        className="absolute left-0 top-full z-50 mt-2 w-52 overflow-hidden border p-1"
        style={{
          backgroundColor: 'var(--panel)',
          borderColor: 'var(--line-card)',
          borderRadius: 'var(--radius-card)',
          boxShadow: 'var(--shadow-panel)',
        }}
      >
        {links.map(item => (
          <a
            key={item.href}
            href={item.href}
            onClick={() => {
              setOpen(false)
              onNavigate?.()
            }}
            className="block rounded px-3 py-2 text-[14px] font-medium no-underline"
            style={{ color: 'var(--ink-nav)' }}
          >
            {item.label}
          </a>
        ))}
      </div>
    </div>
  )
}

export function Header() {
  const [menuOpen, setMenuOpen] = useState(false)
  const [mobileAudiencesOpen, setMobileAudiencesOpen] = useState(false)

  useEffect(() => {
    document.body.style.overflow = menuOpen ? 'hidden' : ''
    return () => {
      document.body.style.overflow = ''
    }
  }, [menuOpen])

  const closeMenu = () => {
    setMenuOpen(false)
    setMobileAudiencesOpen(false)
  }

  return (
    <>
      <header
        className="sticky top-0 z-50 border-b backdrop-blur-[10px]"
        style={{ backgroundColor: 'rgba(251,246,236,0.82)', borderColor: 'rgba(38,58,60,0.08)' }}
      >
        <div className="mx-auto flex h-[72px] max-w-[1200px] items-center justify-between px-5 md:px-10 xl:px-14">
          <Logo />

          <nav className="hidden items-center gap-7 md:flex" aria-label="Primary">
            <NavDropdown label="Platform" links={PLATFORM_LINKS} />
            <NavDropdown label="Solutions" links={AUDIENCE_LINKS} />
            <NavDropdown label="Resources" links={RESOURCE_LINKS} />
            {NAV_LINKS.map(({ label, href }) => (
              <a
                key={href}
                href={href}
                className="text-[15px] font-medium no-underline transition-colors"
                style={{ color: 'var(--text)' }}
                onMouseEnter={e => {
                  e.currentTarget.style.color = 'var(--ink-nav)'
                }}
                onMouseLeave={e => {
                  e.currentTarget.style.color = 'var(--text)'
                }}
              >
                {label}
              </a>
            ))}
          </nav>

          <div className="flex items-center gap-3">
            <LocaleSwitcher compact />
            <a href="/get-started" className="btn-nav-cta hidden sm:inline-flex">
              Get started
            </a>
            <button
              type="button"
              onClick={() => setMenuOpen(true)}
              data-nav-menu-open
              className="inline-flex h-9 w-9 cursor-pointer items-center justify-center rounded md:hidden"
              style={{ color: 'var(--ink-nav)' }}
              aria-expanded={menuOpen}
              aria-controls="mobile-nav"
              aria-label="Open menu"
            >
              <Menu className="h-5 w-5" />
            </button>
          </div>
        </div>
      </header>
      <Breadcrumbs />

      <div
        className="fixed inset-0 z-[60] flex flex-col md:hidden"
        id="mobile-nav"
        data-nav-menu
        hidden={!menuOpen}
        role="dialog"
        aria-modal="true"
        aria-label="Navigation"
        style={{ backgroundColor: 'var(--paper)' }}
      >
        <div
          className="flex h-[72px] items-center justify-between border-b px-5"
          style={{ borderColor: 'var(--line)' }}
        >
          <Logo />
          <button
            type="button"
            onClick={closeMenu}
            data-nav-menu-close
            className="inline-flex h-9 w-9 cursor-pointer items-center justify-center rounded"
            style={{ color: 'var(--ink-nav)' }}
            aria-label="Close menu"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <nav className="flex flex-1 flex-col gap-1 overflow-y-auto p-4" aria-label="Mobile primary">
          <div>
            <button
              type="button"
              onClick={() => setMobileAudiencesOpen(v => !v)}
              data-nav-mobile-audiences
              className="flex w-full cursor-pointer items-center justify-between rounded px-3 py-3 text-[15px] font-medium"
              style={{ color: 'var(--ink-nav)' }}
              aria-expanded={mobileAudiencesOpen}
            >
              Solutions
              <ChevronDown
                className={`h-4 w-4 transition-transform duration-150 ${mobileAudiencesOpen ? 'rotate-180' : ''}`}
                aria-hidden
              />
            </button>
            <div
              data-nav-mobile-audiences-panel
              hidden={!mobileAudiencesOpen}
              className="ml-3 mt-1 flex flex-col gap-1 border-l-2 pl-4"
              style={{ borderColor: 'var(--line)' }}
            >
              {AUDIENCE_LINKS.map(item => (
                <a
                  key={item.href}
                  href={item.href}
                  onClick={closeMenu}
                  className="rounded px-3 py-2 text-[14px] no-underline"
                  style={{ color: 'var(--text)' }}
                >
                  {item.label}
                </a>
              ))}
            </div>
          </div>

          <a href="/platform" onClick={closeMenu} className="rounded px-3 py-3 text-[15px] font-medium no-underline" style={{ color: 'var(--ink-nav)' }}>Platform</a>
          {PLATFORM_LINKS.slice(1).map(item => <a key={item.href} href={item.href} onClick={closeMenu} className="ml-4 rounded px-3 py-2 text-[14px] no-underline" style={{ color: 'var(--text)' }}>{item.label}</a>)}
          <a href="/resources" onClick={closeMenu} className="rounded px-3 py-3 text-[15px] font-medium no-underline" style={{ color: 'var(--ink-nav)' }}>Resources</a>
          {RESOURCE_LINKS.slice(1).map(item => <a key={item.href} href={item.href} onClick={closeMenu} className="ml-4 rounded px-3 py-2 text-[14px] no-underline" style={{ color: 'var(--text)' }}>{item.label}</a>)}

          {NAV_LINKS.map(({ label, href }) => (
            <a
              key={href}
              href={href}
              onClick={closeMenu}
              className="rounded px-3 py-3 text-[15px] font-medium no-underline"
              style={{ color: 'var(--ink-nav)' }}
            >
              {label}
            </a>
          ))}
        </nav>

        <div className="border-t p-4 pb-8" style={{ borderColor: 'var(--line)' }}>
          <a href="/get-started" onClick={closeMenu} className="btn-nav-cta w-full justify-center">
            Get started
          </a>
        </div>
      </div>
    </>
  )
}
