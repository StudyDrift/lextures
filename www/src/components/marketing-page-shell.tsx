import type { ReactNode } from 'react'
import { Header } from './header'
import { SiteFooter } from './site-footer'
import { WindLines } from './home/wind-lines'

type MarketingPageShellProps = {
  children: ReactNode
}

export function MarketingPageShell({ children }: MarketingPageShellProps) {
  return (
    <div
      className="min-h-screen overflow-x-hidden antialiased"
      style={{ backgroundColor: 'var(--paper)', color: 'var(--text)' }}
    >
      <Header />
      <main>{children}</main>
      <SiteFooter />
    </div>
  )
}

type AudienceHeroProps = {
  eyebrow: string
  title: string
  lead: string
  primaryHref: string
  primaryLabel: string
  secondaryHref?: string
  secondaryLabel?: string
}

export function AudienceHero({
  eyebrow,
  title,
  lead,
  primaryHref,
  primaryLabel,
  secondaryHref,
  secondaryLabel,
}: AudienceHeroProps) {
  return (
    <section className="relative overflow-hidden">
      <WindLines variant="hero" />
      <div
        className="relative z-[2] mx-auto max-w-[960px] px-5 py-16 md:px-10 md:py-20 xl:px-14"
        style={{ animation: 'lx-fade-up 0.7s ease both' }}
      >
        <span
          className="inline-flex items-center rounded-full px-3.5 py-[7px] text-[13px] font-semibold uppercase tracking-[0.04em]"
          style={{ color: '#4fa894', backgroundColor: 'rgba(106,197,176,0.14)' }}
        >
          {eyebrow}
        </span>
        <h1
          className="font-display mt-5 max-w-[720px] font-semibold leading-[1.05] tracking-[-0.02em] text-balance"
          style={{ color: '#22333b', fontSize: 'clamp(32px,4.4vw,52px)' }}
        >
          {title}
        </h1>
        <p className="mt-6 max-w-[640px] text-[18px] leading-[1.6]" style={{ color: '#4a5b5d' }}>
          {lead}
        </p>
        <div className="mt-8 flex flex-wrap gap-3.5">
          <a href={primaryHref} className="btn-primary">
            {primaryLabel}
          </a>
          {secondaryHref && secondaryLabel && (
            <a href={secondaryHref} className="btn-secondary">
              {secondaryLabel}
            </a>
          )}
        </div>
      </div>
      <div
        className="h-px"
        style={{ background: 'linear-gradient(90deg, transparent, rgba(38,58,60,0.1), transparent)' }}
      />
    </section>
  )
}

type CardGridProps = {
  heading: string
  subheading?: string
  items: { title: string; body: string }[]
  columns?: 2 | 3
}

const SAIL_PALETTE = [
  { color: '#f2684e', tint: 'rgba(242,104,78,0.12)' },
  { color: '#f49b44', tint: 'rgba(244,155,68,0.14)' },
  { color: '#6ac5b0', tint: 'rgba(106,197,176,0.16)' },
  { color: '#4fa894', tint: 'rgba(79,168,148,0.14)' },
]

function SailIcon({ color }: { color: string }) {
  return (
    <svg viewBox="0 0 48 58" width="26" height="31" aria-hidden>
      <line x1="8" y1="4" x2="8" y2="54" stroke="#22333b" strokeWidth="3.4" strokeLinecap="round" />
      <path d="M11,9 C30,11 37,22 33,32 C29,42 20,46 11,47 Z" fill={color} />
    </svg>
  )
}

export function CardGrid({ heading, subheading, items, columns = 3 }: CardGridProps) {
  const gridClass = columns === 2 ? 'md:grid-cols-2' : 'md:grid-cols-2 xl:grid-cols-3'

  return (
    <section className="py-16 md:py-20">
      <div className="mx-auto max-w-[960px] px-5 md:px-10 xl:px-14">
        <h2
          className="font-display text-[clamp(26px,3.2vw,38px)] font-semibold leading-tight tracking-[-0.015em]"
          style={{ color: '#22333b' }}
        >
          {heading}
        </h2>
        {subheading && (
          <p className="mt-3 max-w-[640px] text-[16px] leading-relaxed" style={{ color: '#56676a' }}>
            {subheading}
          </p>
        )}
        <div className={`mt-10 grid gap-5 ${gridClass}`}>
          {items.map((item, i) => {
            const sail = SAIL_PALETTE[i % SAIL_PALETTE.length]
            return (
              <article
                key={item.title}
                className="min-w-0 rounded-[18px] px-6 pb-7 pt-[26px] shadow-[0_10px_26px_rgba(34,51,59,0.05)] transition-all duration-200 hover:-translate-y-1 hover:shadow-[0_18px_38px_rgba(34,51,59,0.09)]"
                style={{ backgroundColor: 'var(--panel)', border: '1px solid rgba(38,58,60,0.08)' }}
              >
                <div
                  className="flex h-[48px] w-[48px] items-center justify-center rounded-[12px]"
                  style={{ backgroundColor: sail.tint }}
                >
                  <SailIcon color={sail.color} />
                </div>
                <h3
                  className="font-display mt-[18px] text-[20px] font-semibold leading-snug"
                  style={{ color: '#22333b' }}
                >
                  {item.title}
                </h3>
                <p className="mt-2.5 text-[15px] leading-[1.55]" style={{ color: '#56676a' }}>
                  {item.body}
                </p>
              </article>
            )
          })}
        </div>
      </div>
    </section>
  )
}

type TagStripProps = {
  label: string
  tags: string[]
}

export function TagStrip({ label, tags }: TagStripProps) {
  return (
    <section className="py-12">
      <div className="mx-auto max-w-[960px] px-5 md:px-10 xl:px-14">
        <span
          className="text-[13px] font-semibold uppercase tracking-[0.06em]"
          style={{ color: 'var(--coral)' }}
        >
          {label}
        </span>
        <div className="mt-4 flex flex-wrap gap-2.5">
          {tags.map(tag => (
            <span
              key={tag}
              className="rounded-full px-3.5 py-2 text-[14px] font-medium"
              style={{
                color: '#47585a',
                backgroundColor: 'var(--panel)',
                border: '1px solid rgba(38,58,60,0.1)',
              }}
            >
              {tag}
            </span>
          ))}
        </div>
      </div>
    </section>
  )
}

type AudienceCtaProps = {
  title: string
  body: string
  primaryHref: string
  primaryLabel: string
  secondaryHref?: string
  secondaryLabel?: string
}

export function AudienceCta({
  title,
  body,
  primaryHref,
  primaryLabel,
  secondaryHref,
  secondaryLabel,
}: AudienceCtaProps) {
  return (
    <section className="relative overflow-hidden" style={{ backgroundColor: '#4fa894', color: '#fff' }}>
      <WindLines variant="teal" />
      <div className="relative z-[2] mx-auto max-w-[720px] px-5 py-20 text-center md:px-10 md:py-24">
        <h2
          className="font-display font-semibold leading-[1.06] tracking-[-0.02em]"
          style={{ fontSize: 'clamp(28px,3.6vw,44px)' }}
        >
          {title}
        </h2>
        <p className="mt-4 text-[17px] leading-[1.55]" style={{ color: 'rgba(255,255,255,0.9)' }}>
          {body}
        </p>
        <div className="mt-8 flex flex-col items-center justify-center gap-3.5 sm:flex-row">
          <a href={primaryHref} className="btn-primary">
            {primaryLabel}
          </a>
          {secondaryHref && secondaryLabel && (
            <a
              href={secondaryHref}
              className="inline-flex items-center gap-2 rounded-full px-6 py-3.5 text-[16px] font-semibold text-white no-underline transition-colors duration-150"
              style={{
                backgroundColor: 'rgba(255,255,255,0.14)',
                border: '1px solid rgba(255,255,255,0.3)',
              }}
              onMouseEnter={e => {
                e.currentTarget.style.backgroundColor = 'rgba(255,255,255,0.22)'
              }}
              onMouseLeave={e => {
                e.currentTarget.style.backgroundColor = 'rgba(255,255,255,0.14)'
              }}
            >
              {secondaryLabel}
            </a>
          )}
        </div>
      </div>
    </section>
  )
}
