import type { ReactNode } from 'react'
import { Header } from './header'
import { SiteFooter } from './site-footer'

type MarketingPageShellProps = {
  children: ReactNode
}

export function MarketingPageShell({ children }: MarketingPageShellProps) {
  return (
    <div className="min-h-screen antialiased" style={{ backgroundColor: 'var(--paper)', color: 'var(--text)' }}>
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
    <section className="border-b" style={{ borderColor: 'var(--line)' }}>
      <div className="mx-auto max-w-[960px] px-5 py-16 md:px-10 md:py-20 xl:px-14">
        <p className="eyebrow-label">{eyebrow}</p>
        <h1
          className="font-display mt-4 max-w-[720px] text-[clamp(32px,4vw,48px)] font-semibold leading-[1.08] tracking-[-0.02em]"
          style={{ color: 'var(--ink)' }}
        >
          {title}
        </h1>
        <p className="mt-6 max-w-[640px] text-[18px] leading-[1.6]" style={{ color: 'var(--text-soft)' }}>
          {lead}
        </p>
        <div className="mt-8 flex flex-wrap gap-3">
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
    </section>
  )
}

type CardGridProps = {
  heading: string
  subheading?: string
  items: { title: string; body: string }[]
  columns?: 2 | 3
}

export function CardGrid({ heading, subheading, items, columns = 3 }: CardGridProps) {
  const gridClass = columns === 2 ? 'md:grid-cols-2' : 'md:grid-cols-2 xl:grid-cols-3'

  return (
    <section className="border-b py-16 md:py-20" style={{ borderColor: 'var(--line)' }}>
      <div className="mx-auto max-w-[960px] px-5 md:px-10 xl:px-14">
        <h2 className="font-display text-[clamp(26px,3vw,34px)] font-semibold leading-tight" style={{ color: 'var(--ink)' }}>
          {heading}
        </h2>
        {subheading && (
          <p className="mt-3 max-w-[640px] text-[16px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
            {subheading}
          </p>
        )}
        <div className={`mt-10 grid gap-5 ${gridClass}`}>
          {items.map(item => (
            <article
              key={item.title}
              className="border p-6"
              style={{
                backgroundColor: 'var(--panel)',
                borderColor: 'var(--line-card)',
                borderRadius: 'var(--radius-card)',
              }}
            >
              <h3 className="text-[17px] font-semibold leading-snug" style={{ color: 'var(--ink-nav)' }}>
                {item.title}
              </h3>
              <p className="mt-3 text-[15px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
                {item.body}
              </p>
            </article>
          ))}
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
    <section className="border-b py-12" style={{ borderColor: 'var(--line)' }}>
      <div className="mx-auto max-w-[960px] px-5 md:px-10 xl:px-14">
        <p className="section-label">{label}</p>
        <div className="mt-4 flex flex-wrap gap-x-5 gap-y-2">
          {tags.map(tag => (
            <span key={tag} className="text-[14px] font-medium" style={{ color: 'var(--text-soft)' }}>
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
    <section className="py-16 md:py-20">
      <div className="mx-auto max-w-[640px] px-5 text-center md:px-10">
        <h2 className="font-display text-[clamp(26px,3vw,34px)] font-semibold leading-tight" style={{ color: 'var(--ink)' }}>
          {title}
        </h2>
        <p className="mt-4 text-[16px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
          {body}
        </p>
        <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
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
    </section>
  )
}
