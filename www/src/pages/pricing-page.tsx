import type { ReactNode } from 'react'
import { ArrowRight, Check } from 'lucide-react'
import { MarketingPageShell } from '../components/marketing-page-shell'
import { WindLines } from '../components/home/wind-lines'
import { SITE_LINKS } from '../lib/site-links'

const SELF_HOST_FEATURES = [
  'Full source under AGPL-3.0 — no license fees',
  'No enforced student or course limits in the software',
  'LTI 1.3, SAML/OIDC, SCIM, Canvas import, QTI',
  'Adaptive quizzes, spaced repetition, grade audit log',
  'iOS and Android apps connect to your instance',
  'You choose which platform features to enable',
]

const SELF_LEARNER_FEATURES = [
  'Adaptive quizzes and spaced-repetition review',
  'Self-paced courses with progress and what-if grades',
  'Create your own courses or enroll in shared catalogs',
  'Web, iOS, and Android — one account across devices',
  'Optional AI-assisted study when enabled on the instance',
]

const INSTITUTION_FEATURES = [
  'Hosted or managed deployment on your domain',
  'SAML/OIDC single sign-on and SCIM 2.0 provisioning',
  'Clever and ClassLink for K–12 identity when needed',
  'Multi-school rollouts, blueprints, and grade audit logs',
  'LTI 1.3 inside Canvas, Moodle, or Blackboard',
  'Implementation and production support options',
]

const INSTITUTION_NOTES = [
  {
    title: 'How pricing works',
    body: 'Quotes depend on enrollment, hosting model, and support level. There is no fixed per-seat price on this page because district and university deployments vary widely.',
  },
  {
    title: 'What you get',
    body: 'Production hosting (or help running your own stack), SSO and roster integration, platform feature configuration, and optional onboarding for instructors and registrars.',
  },
  {
    title: 'Pilot and evaluation',
    body: 'Many teams evaluate flows on a sandbox first, then move to a named institution account when SSO, data residency, and support requirements are clear.',
  },
]

const FAQS = [
  {
    q: 'Is self-hosting really free?',
    a: 'Yes. The software is open source under AGPL-3.0. You pay for your own servers, Postgres, and optional AI API keys — not per-seat licensing to us.',
  },
  {
    q: 'Do all features work out of the box when I self-host?',
    a: 'The codebase includes K–12, higher-ed, and self-learner capabilities. Some surfaces (parent portal, public catalog, Stripe billing) are controlled by platform feature flags your admin enables in Settings → Global platform.',
  },
  {
    q: 'How do university and district accounts work?',
    a: 'Institutions receive a dedicated environment — hosted by us or on infrastructure you control — with SSO, provisioning, and support scoped to your rollout. Use the request information form on the pricing page to describe your enrollment and integration needs.',
  },
  {
    q: 'How do self-learner accounts work?',
    a: 'Sign up at self.lextures.com for a hosted individual account with adaptive practice and spaced-repetition review. You can also self-host the stack for free or use an institution account if your school provides access.',
  },
  {
    q: 'Does AI cost extra?',
    a: 'AI-assisted question generation, tutoring, and grading require an OpenRouter API key configured in your instance. The LMS works without AI; adaptive IRT and spaced repetition do not require it.',
  },
  {
    q: 'Can we use Lextures inside Canvas or Moodle?',
    a: 'Yes. Lextures implements LTI 1.3 as both a tool provider (embed in another LMS) and a platform consumer (launch external tools). Grade passback uses AGS.',
  },
  {
    q: 'What about mobile apps?',
    a: 'Native iOS and Android apps in the repo connect to your API. Students, instructors, and parents (when the parent portal is enabled) use the same backend as the web app.',
  },
]

type PricingCardProps = {
  label: string
  price: ReactNode
  description: string
  features: string[]
  cta: ReactNode
  muted?: boolean
  badge?: string
}

function PricingCard({ label, price, description, features, cta, muted, badge }: PricingCardProps) {
  return (
    <div
      className={`relative flex h-full flex-col rounded-[18px] p-8 ${muted ? 'opacity-70' : ''}`}
      style={{
        backgroundColor: muted ? 'var(--panel-sunken)' : 'var(--panel)',
        border: '1px solid rgba(38,58,60,0.08)',
        boxShadow: muted ? undefined : '0 14px 34px rgba(34,51,59,0.07)',
      }}
    >
      {badge && (
        <span
          className="absolute right-6 top-6 rounded-full px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-[0.12em]"
          style={{ backgroundColor: 'rgba(106,197,176,0.16)', color: '#4fa894' }}
        >
          {badge}
        </span>
      )}
      <span
        className="text-[13px] font-semibold uppercase tracking-[0.06em]"
        style={{ color: 'var(--coral)' }}
      >
        {label}
      </span>
      <div className="mt-3">{price}</div>
      <p className="mt-4 text-[15px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
        {description}
      </p>
      <ul className="mt-6 flex-1 space-y-3">
        {features.map(feature => (
          <li key={feature} className="flex items-start gap-2.5">
            <Check className="mt-0.5 h-4 w-4 shrink-0" style={{ color: 'var(--teal-deep)' }} aria-hidden />
            <span className="text-[14px]" style={{ color: 'var(--text)' }}>
              {feature}
            </span>
          </li>
        ))}
      </ul>
      <div className="mt-8">{cta}</div>
    </div>
  )
}

export function PricingPage() {
  return (
    <MarketingPageShell>
      <section className="relative overflow-hidden">
        <WindLines variant="hero" />
        <div className="relative z-[2] mx-auto max-w-[1100px] px-5 py-16 text-center md:px-10 md:py-20 xl:px-14">
          <span
            className="inline-flex items-center rounded-full px-3.5 py-[7px] text-[13px] font-semibold uppercase tracking-[0.04em]"
            style={{ color: '#4fa894', backgroundColor: 'rgba(106,197,176,0.14)' }}
          >
            Pricing
          </span>
          <h1
            className="font-display mx-auto mt-5 max-w-[860px] font-semibold leading-[1.05] tracking-[-0.02em] text-balance"
            style={{ color: '#22333b', fontSize: 'clamp(32px,4.4vw,52px)' }}
          >
            Self-host for free. Pay only for infrastructure you choose.
          </h1>
          <p className="mx-auto mt-5 max-w-[680px] text-[18px] leading-relaxed" style={{ color: '#4a5b5d' }}>
            Lextures is AGPL-3.0 open source. Run it yourself on Postgres, open a university or
            district account for production hosting and support, or sign up as an independent
            learner at self.lextures.com.
          </p>
        </div>
        <div
          className="h-px"
          style={{ background: 'linear-gradient(90deg, transparent, rgba(38,58,60,0.1), transparent)' }}
        />
      </section>

      <section className="py-16 md:py-20">
        <div className="mx-auto max-w-[1100px] px-5 md:px-10 xl:px-14">
          <div className="grid gap-8 lg:grid-cols-3">
            <PricingCard
              label="Self-host"
              price={
                <div className="flex items-baseline gap-2">
                  <span className="font-display text-5xl font-semibold" style={{ color: 'var(--ink)' }}>
                    $0
                  </span>
                  <span className="text-[15px]" style={{ color: 'var(--text-soft)' }}>
                    license fees
                  </span>
                </div>
              }
              description="Clone the repo, run Docker Compose, and operate on your Postgres. Typical for universities, districts, and developers who want full control."
              features={SELF_HOST_FEATURES}
              cta={
                <a href={SITE_LINKS.github} className="btn-primary w-full justify-center">
                  View on GitHub
                </a>
              }
            />

            <PricingCard
              label="Self-learner"
              price={
                <p className="font-display text-2xl font-semibold" style={{ color: 'var(--ink)' }}>
                  Individual accounts
                </p>
              }
              description="For certification prep, language study, and independent learners who want adaptive practice without running their own server."
              features={SELF_LEARNER_FEATURES}
              cta={
                <a href={SITE_LINKS.selfLearner} className="btn-primary w-full justify-center">
                  Sign up
                </a>
              }
            />

            <PricingCard
              label="University or district"
              price={
                <p className="font-display text-2xl font-semibold" style={{ color: 'var(--ink)' }}>
                  Custom quote
                </p>
              }
              description="Production accounts for colleges, universities, and K–12 districts — hosted by Lextures or deployed on infrastructure you designate."
              features={INSTITUTION_FEATURES}
              cta={
                <a href="/request-information" className="btn-secondary w-full justify-center">
                  Request information
                </a>
              }
            />
          </div>

          <div
            className="mt-10 rounded-[18px] p-8"
            style={{
              backgroundColor: '#f1e5c6',
              border: '1px solid rgba(201,168,106,0.35)',
            }}
          >
            <h2 className="font-display text-[clamp(22px,2.5vw,28px)] font-semibold" style={{ color: 'var(--ink)' }}>
              University and district accounts
            </h2>
            <p className="mt-3 max-w-[720px] text-[15px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
              Institutional accounts are scoped to your organization: roster provisioning, SSO,
              feature flags, and support aligned to how your registrar and IT team operate — not a
              shared public sandbox.
            </p>
            <div className="mt-6 grid gap-6 md:grid-cols-3">
              {INSTITUTION_NOTES.map(note => (
                <div key={note.title}>
                  <p className="text-[15px] font-semibold" style={{ color: 'var(--ink-nav)' }}>
                    {note.title}
                  </p>
                  <p className="mt-1 text-[14px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
                    {note.body}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="py-16 md:py-20">
        <div className="mx-auto max-w-[720px] px-5 md:px-10 xl:px-14">
          <h2 className="font-display text-[clamp(26px,3vw,34px)] font-semibold leading-tight tracking-[-0.015em]" style={{ color: '#22333b' }}>
            Common questions
          </h2>
          <dl className="mt-8 divide-y" style={{ borderColor: 'var(--line)' }}>
            {FAQS.map(({ q, a }) => (
              <div key={q} className="py-6">
                <dt className="font-semibold" style={{ color: 'var(--ink-nav)' }}>
                  {q}
                </dt>
                <dd className="mt-2 text-[15px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
                  {a}
                </dd>
              </div>
            ))}
          </dl>
        </div>
      </section>

      <section className="relative overflow-hidden" style={{ backgroundColor: '#4fa894', color: '#fff' }}>
        <WindLines variant="teal" />
        <div className="relative z-[2] mx-auto max-w-[720px] px-5 py-20 text-center md:px-10 md:py-24">
          <h2
            className="font-display font-semibold leading-[1.06] tracking-[-0.02em]"
            style={{ fontSize: 'clamp(28px,3.6vw,44px)' }}
          >
            Start with self-host, scale to your campus
          </h2>
          <p className="mt-4 text-[17px] leading-[1.55]" style={{ color: 'rgba(255,255,255,0.9)' }}>
            Clone the repo and run a pilot on your Postgres today. When you need SSO, managed
            hosting, or district-wide rollout, request a university or district account.
          </p>
          <div className="mt-8 flex flex-col items-center justify-center gap-3.5 sm:flex-row">
            <a href="/get-started" className="btn-primary gap-2">
              Get started
              <ArrowRight className="h-4 w-4" aria-hidden />
            </a>
            <a
              href="/docs/self-hosting"
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
              Self-hosting guide
            </a>
          </div>
        </div>
      </section>
    </MarketingPageShell>
  )
}
