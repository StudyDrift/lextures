import { ChevronDown, Github, Menu, X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

const LINKS = {
  demo: 'https://demo.lextures.com/',
  github: 'https://github.com/StudyDrift/lextures',
} as const

const INDUSTRIES = [
  { label: 'Higher Education', href: '/higher-ed' },
  { label: 'K–12', href: '/k-12' },
  { label: 'Self-Learner', href: '/self-learner' },
]

function Logo() {
  return (
    <a href="/" className="flex items-center gap-2 no-underline" aria-label="Lextures home">
      <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-accent text-white shadow-sm">
        <img src="/logo.svg" className="h-4 w-4 brightness-0 invert" alt="" aria-hidden />
      </span>
      <span className="text-[0.9375rem] font-semibold tracking-tight text-slate-900">Lextures</span>
    </a>
  )
}

function IndustriesDropdown({ onNavigate }: { onNavigate?: () => void }) {
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
        className="flex cursor-pointer items-center gap-1 rounded-lg px-3 py-1.5 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900"
      >
        Solutions
        <ChevronDown
          className={`h-3.5 w-3.5 transition-transform duration-150 ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div className="absolute left-0 top-full z-50 mt-2 w-48 overflow-hidden rounded-xl border border-slate-200 bg-white p-1 shadow-xl shadow-slate-900/10">
          {INDUSTRIES.map(item => (
            <a
              key={item.href}
              href={item.href}
              onClick={() => { setOpen(false); onNavigate?.() }}
              className="block rounded-lg px-3 py-2 text-sm font-medium text-slate-700 no-underline transition-colors hover:bg-slate-50 hover:text-slate-900"
            >
              {item.label}
            </a>
          ))}
        </div>
      )}
    </div>
  )
}

const NAV_LINKS = [
  { label: 'Pricing', href: '/pricing' },
  { label: 'Blog', href: '/blog' },
  { label: 'Docs', href: '/docs' },
]

export function Header() {
  const [menuOpen, setMenuOpen] = useState(false)
  const [mobileIndustriesOpen, setMobileIndustriesOpen] = useState(false)

  useEffect(() => {
    document.body.style.overflow = menuOpen ? 'hidden' : ''
    return () => { document.body.style.overflow = '' }
  }, [menuOpen])

  const closeMenu = () => {
    setMenuOpen(false)
    setMobileIndustriesOpen(false)
  }

  return (
    <>
      <header className="sticky top-0 z-50 px-3 pt-3 pb-0 sm:px-4">
        <div className="mx-auto max-w-6xl">
          <div className="flex h-[52px] items-center justify-between rounded-2xl border border-slate-200/80 bg-white/95 px-3 shadow-sm shadow-slate-900/5 backdrop-blur-xl sm:px-4">
            <Logo />

            {/* Desktop nav */}
            <nav className="hidden items-center gap-0.5 md:flex" aria-label="Primary">
              <IndustriesDropdown />
              {NAV_LINKS.map(({ label, href }) => (
                <a
                  key={href}
                  href={href}
                  className="rounded-lg px-3 py-1.5 text-sm font-medium text-slate-600 no-underline transition-colors hover:bg-slate-100 hover:text-slate-900"
                >
                  {label}
                </a>
              ))}
            </nav>

            {/* Desktop actions */}
            <div className="hidden items-center gap-2 md:flex">
              <a
                href={LINKS.demo}
                className="rounded-lg px-3 py-1.5 text-sm font-medium text-slate-600 no-underline transition-colors hover:bg-slate-100 hover:text-slate-900"
                target="_blank"
                rel="noopener noreferrer"
              >
                Live demo
              </a>
              <a
                href={LINKS.github}
                className="rounded-lg p-1.5 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900"
                aria-label="View on GitHub"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Github className="h-4 w-4" />
              </a>
              <a href="/get-started" className="btn-primary py-2 text-xs">
                Get Started
              </a>
            </div>

            {/* Mobile menu button */}
            <button
              type="button"
              onClick={() => setMenuOpen(true)}
              className="inline-flex h-8 w-8 cursor-pointer items-center justify-center rounded-lg border border-slate-200 text-slate-600 transition hover:bg-slate-100 md:hidden"
              aria-expanded={menuOpen}
              aria-controls="mobile-nav"
              aria-label="Open menu"
            >
              <Menu className="h-4 w-4" />
            </button>
          </div>
        </div>
      </header>

      {/* Mobile overlay */}
      {menuOpen && (
        <div
          className="fixed inset-0 z-[60] flex flex-col bg-white md:hidden"
          id="mobile-nav"
          role="dialog"
          aria-modal="true"
          aria-label="Navigation"
        >
          <div className="flex h-[52px] items-center justify-between border-b border-slate-100 px-4">
            <Logo />
            <button
              type="button"
              onClick={closeMenu}
              className="inline-flex h-8 w-8 cursor-pointer items-center justify-center rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-100"
              aria-label="Close menu"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          <nav className="flex flex-1 flex-col gap-1 overflow-y-auto p-3" aria-label="Mobile primary">
            <div>
              <button
                type="button"
                onClick={() => setMobileIndustriesOpen(v => !v)}
                className="flex w-full cursor-pointer items-center justify-between rounded-xl px-4 py-3 text-sm font-medium text-slate-800 transition hover:bg-slate-50"
                aria-expanded={mobileIndustriesOpen}
              >
                Solutions
                <ChevronDown
                  className={`h-4 w-4 transition-transform duration-150 ${mobileIndustriesOpen ? 'rotate-180' : ''}`}
                  aria-hidden
                />
              </button>
              {mobileIndustriesOpen && (
                <div className="ml-4 mt-1 flex flex-col gap-1 border-l-2 border-slate-100 pl-4">
                  {INDUSTRIES.map(item => (
                    <a
                      key={item.href}
                      href={item.href}
                      onClick={closeMenu}
                      className="rounded-lg px-3 py-2.5 text-sm font-medium text-slate-700 no-underline transition hover:bg-slate-50 hover:text-slate-900"
                    >
                      {item.label}
                    </a>
                  ))}
                </div>
              )}
            </div>

            {NAV_LINKS.map(({ label, href }) => (
              <a
                key={href}
                href={href}
                onClick={closeMenu}
                className="rounded-xl px-4 py-3 text-sm font-medium text-slate-800 no-underline transition hover:bg-slate-50"
              >
                {label}
              </a>
            ))}
          </nav>

          <div className="flex flex-col gap-2 border-t border-slate-100 p-4 pb-8">
            <a href="/get-started" onClick={closeMenu} className="btn-primary w-full justify-center py-3">
              Get Started
            </a>
            <a
              href={LINKS.demo}
              onClick={closeMenu}
              className="btn-secondary w-full justify-center py-3"
              target="_blank"
              rel="noopener noreferrer"
            >
              Live Demo
            </a>
            <a
              href={LINKS.github}
              onClick={closeMenu}
              className="flex cursor-pointer items-center justify-center gap-2 rounded-xl px-4 py-2.5 text-sm font-medium text-slate-600 no-underline transition hover:bg-slate-50"
              target="_blank"
              rel="noopener noreferrer"
            >
              <Github className="h-4 w-4" />
              View on GitHub
            </a>
          </div>
        </div>
      )}
    </>
  )
}
