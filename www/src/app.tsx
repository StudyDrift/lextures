import {
  ArrowRight,
  BarChart3,
  BookOpen,
  BrainCircuit,
  Code2,
  GraduationCap,
  RefreshCw,
  ShieldCheck,
  Unplug,
  Zap,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { Header } from './components/header'
import { BlogIndex } from './pages/blog-index'
import { BlogPost } from './pages/blog-post'
import { DocsIndex } from './pages/docs-index'
import { DocsPost } from './pages/docs-post'
import { GetStartedPage } from './pages/get-started-page'
import { HigherEdPage } from './pages/higher-ed-page'
import { K12Page } from './pages/k12-page'
import { PricingPage } from './pages/pricing-page'
import { SelfLearnerPage } from './pages/self-learner-page'
import {
  PrivacyPolicyHistoryPage,
  PrivacyPolicyPage,
  TermsOfServiceHistoryPage,
  TermsOfServicePage,
} from './pages/legal-pages'
import { SecurityPage } from './pages/security-page'
import { AccessibilityConformancePage } from './pages/accessibility-conformance-page'
import { CaliforniaPrivacyRightsPage } from './pages/california-privacy-rights-page'
import { VpatPage } from './pages/vpat-page'
import { SiteFooter } from './components/site-footer'

const LINKS = {
  demo: 'https://demo.lextures.com/',
  github: 'https://github.com/StudyDrift/lextures',
} as const

function useHashRoute() {
  const [hash, setHash] = useState(() => window.location.hash)
  useEffect(() => {
    const handler = () => setHash(window.location.hash)
    window.addEventListener('hashchange', handler)
    return () => window.removeEventListener('hashchange', handler)
  }, [])
  return hash
}

/* ─────────────────────────────────────────────────────────
   HOMEPAGE
   ───────────────────────────────────────────────────────── */
function HomePage() {
  useEffect(() => {
    const hash = window.location.hash
    if (hash && !hash.startsWith('#/')) {
      document.querySelector(hash)?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [])

  return (
    <div className="min-h-screen bg-white text-slate-900 antialiased">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[100] focus:rounded-xl focus:bg-accent focus:px-4 focus:py-2 focus:text-sm focus:text-white"
      >
        Skip to content
      </a>

      <Header />

      <main id="main">
        {/* ═══════════════════════════════════════════ HERO ═══ */}
        <section className="relative overflow-hidden bg-[#020617] pb-0 pt-16 sm:pt-20">
          {/* Gradient orbs */}
          <div
            aria-hidden
            className="pointer-events-none absolute -left-48 -top-24 h-[600px] w-[600px] rounded-full bg-indigo-600/25 blur-[120px]"
          />
          <div
            aria-hidden
            className="pointer-events-none absolute -right-32 bottom-0 h-[500px] w-[500px] rounded-full bg-violet-600/20 blur-[100px]"
          />
          <div
            aria-hidden
            className="pointer-events-none absolute left-1/2 top-1/3 h-[300px] w-[300px] -translate-x-1/2 rounded-full bg-indigo-500/10 blur-[80px]"
          />

          {/* Grid pattern */}
          <div
            aria-hidden
            className="pointer-events-none absolute inset-0"
            style={{
              backgroundImage:
                'linear-gradient(to right, rgba(99,102,241,0.06) 1px, transparent 1px), linear-gradient(to bottom, rgba(99,102,241,0.06) 1px, transparent 1px)',
              backgroundSize: '48px 48px',
              maskImage: 'linear-gradient(to bottom, transparent 0%, black 15%, black 75%, transparent 100%)',
              WebkitMaskImage: 'linear-gradient(to bottom, transparent 0%, black 15%, black 75%, transparent 100%)',
            }}
          />

          <div className="relative z-10 mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            {/* Eyebrow */}
            <div className="flex justify-center">
              <div className="inline-flex items-center gap-2 rounded-full border border-indigo-500/30 bg-indigo-500/10 px-4 py-1.5 backdrop-blur-sm">
                <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-indigo-400" aria-hidden />
                <span className="text-[0.7rem] font-semibold uppercase tracking-[0.22em] text-indigo-300">
                  Adaptive learning management
                </span>
              </div>
            </div>

            {/* Headline */}
            <h1 className="mt-8 text-center text-[clamp(2.5rem,7vw,5.5rem)] font-bold leading-[1.02] tracking-[-0.03em] text-white">
              The LMS that{' '}
              <span className="font-display bg-gradient-to-r from-indigo-400 via-violet-400 to-purple-400 bg-clip-text font-normal italic text-transparent">
                adapts
              </span>
            </h1>

            <p className="mx-auto mt-6 max-w-2xl text-center text-lg leading-relaxed text-slate-400 sm:text-xl">
              Adaptive quizzes, institutional workflows, and integrations built for schools and
              programs running at real scale—not a slide deck with a gradebook attached.
            </p>

            {/* CTAs */}
            <div className="mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row">
              <a href="/get-started" className="btn-primary h-12 gap-2 px-8 text-[0.9375rem]">
                Get Started free
                <ArrowRight className="h-4 w-4" aria-hidden />
              </a>
              <a href={LINKS.demo} className="btn-ghost-dark h-12 px-8 text-[0.9375rem]" target="_blank" rel="noopener noreferrer">
                Live demo
              </a>
            </div>

            <p className="mt-3 text-center text-xs text-slate-500">
              Free tier included · No credit card required · AGPL-3.0 open source
            </p>

            {/* Stats row */}
            <div className="mt-16 grid grid-cols-2 divide-x divide-slate-800 border-t border-slate-800 sm:grid-cols-4">
              {[
                { value: '$0', label: 'to get started' },
                { value: '14+', label: 'question types' },
                { value: 'IRT', label: 'adaptive engine' },
                { value: 'K–12 & HE', label: 'institution ready' },
              ].map(({ value, label }) => (
                <div key={value} className="flex flex-col items-center py-6 text-center">
                  <span className="text-2xl font-bold text-white">{value}</span>
                  <span className="mt-1 text-[0.7rem] font-medium uppercase tracking-wider text-slate-500">
                    {label}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ══════════════════════════════════ FEATURES BENTO ═══ */}
        <section id="features" className="bg-white py-20 sm:py-28">
          <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <div className="mb-12 max-w-xl">
              <p className="text-[0.7rem] font-semibold uppercase tracking-[0.22em] text-indigo-500">
                Platform
              </p>
              <h2 className="mt-3 text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl">
                Everything a serious institution needs
              </h2>
              <p className="mt-4 text-lg leading-relaxed text-slate-500">
                Course management, adaptive assessments, gradebook, and integrations—without
                treating adaptation like a marketing bolt-on.
              </p>
            </div>

            {/* Bento grid */}
            <div className="grid grid-cols-1 gap-4 md:grid-cols-12">

              {/* Card 1 — Large dark, Adaptive quiz */}
              <article className="group relative overflow-hidden rounded-3xl bg-slate-950 p-7 md:col-span-7">
                <div
                  aria-hidden
                  className="pointer-events-none absolute -right-16 -top-16 h-48 w-48 rounded-full bg-indigo-600/30 blur-[60px] transition-all duration-500 group-hover:bg-indigo-500/40"
                />
                <div className="relative z-10">
                  <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-indigo-500/20 text-indigo-400 ring-1 ring-indigo-500/30">
                    <BrainCircuit className="h-5 w-5" aria-hidden />
                  </div>
                  <h3 className="mt-5 text-xl font-semibold text-white">Adaptive quiz delivery</h3>
                  <p className="mt-3 max-w-sm text-sm leading-relaxed text-slate-400">
                    Quizzes adjust difficulty in real time using Item Response Theory. Every learner
                    gets the right questions at the right moment—not a one-size-fits-all test.
                  </p>
                  <div className="mt-6 inline-flex items-center gap-1.5 text-xs font-semibold text-indigo-400">
                    IRT 2PL / 3PL models <ArrowRight className="h-3 w-3" aria-hidden />
                  </div>
                </div>
              </article>

              {/* Card 2 — AI content */}
              <article className="group relative overflow-hidden rounded-3xl border border-slate-200 bg-white p-7 md:col-span-5">
                <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-violet-50 text-violet-600 ring-1 ring-violet-200">
                  <Zap className="h-5 w-5" aria-hidden />
                </div>
                <h3 className="mt-5 text-lg font-semibold text-slate-900">AI-generated content</h3>
                <p className="mt-2.5 text-sm leading-relaxed text-slate-500">
                  Generate quiz questions from learning objectives, build rubrics from assignment
                  descriptions, and produce progressive hints that guide without giving answers.
                </p>
              </article>

              {/* Card 3 — 14+ types, accent background */}
              <article className="relative overflow-hidden rounded-3xl bg-indigo-600 p-7 md:col-span-4">
                <div className="absolute right-4 top-4 text-[3.5rem] font-black leading-none text-white/20 select-none">14+</div>
                <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-white/15 text-white ring-1 ring-white/20">
                  <BookOpen className="h-5 w-5" aria-hidden />
                </div>
                <h3 className="mt-5 text-lg font-semibold text-white">Question types</h3>
                <p className="mt-2 text-sm leading-relaxed text-indigo-200">
                  Multiple choice, essay, live code execution, image hotspots, matching, ordering,
                  formula, and audio/video responses.
                </p>
              </article>

              {/* Card 4 — Gradebook */}
              <article className="rounded-3xl border border-slate-200 bg-slate-50 p-7 md:col-span-4">
                <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-emerald-50 text-emerald-600 ring-1 ring-emerald-200">
                  <BarChart3 className="h-5 w-5" aria-hidden />
                </div>
                <h3 className="mt-5 text-lg font-semibold text-slate-900">Standards-based gradebook</h3>
                <p className="mt-2 text-sm leading-relaxed text-slate-500">
                  Map assignments to NGSS, CCSS, or your own standards. Track mastery by
                  objective—not just points.
                </p>
              </article>

              {/* Card 5 — LTI */}
              <article className="rounded-3xl border border-slate-200 bg-white p-7 md:col-span-4">
                <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-sky-50 text-sky-600 ring-1 ring-sky-200">
                  <Unplug className="h-5 w-5" aria-hidden />
                </div>
                <h3 className="mt-5 text-lg font-semibold text-slate-900">Canvas, Moodle & Blackboard-ready</h3>
                <p className="mt-2 text-sm leading-relaxed text-slate-500">
                  LTI 1.3 provider: run Lextures inside any LMS your institution already uses.
                  Import courses and question banks from Canvas.
                </p>
              </article>

              {/* Card 6 — Wide, enterprise identity */}
              <article className="flex flex-col gap-6 rounded-3xl border border-slate-200 bg-white p-7 md:col-span-12 md:flex-row md:items-center md:gap-12">
                <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-rose-50 text-rose-600 ring-1 ring-rose-200">
                  <ShieldCheck className="h-5 w-5" aria-hidden />
                </div>
                <div className="flex-1">
                  <h3 className="text-lg font-semibold text-slate-900">Enterprise identity & provisioning</h3>
                  <p className="mt-1.5 text-sm leading-relaxed text-slate-500">
                    SAML 2.0, OIDC, Clever, and ClassLink SSO. OneRoster 1.2 CSV and SCIM 2.0 HTTP for bulk roster sync.
                    TOTP and WebAuthn MFA for every account.
                  </p>
                </div>
                <div className="flex flex-wrap gap-2 md:justify-end">
                  {['SAML 2.0', 'OIDC', 'Clever', 'ClassLink', 'SCIM 2.0'].map(tag => (
                    <span key={tag} className="rounded-lg border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-medium text-slate-600">
                      {tag}
                    </span>
                  ))}
                </div>
              </article>

            </div>
          </div>
        </section>

        {/* ════════════════════════════════ ADAPTIVE SCIENCE ═══ */}
        <section id="ai" className="bg-slate-950 py-20 sm:py-28">
          <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <div className="max-w-2xl">
              <p className="text-[0.7rem] font-semibold uppercase tracking-[0.22em] text-indigo-400">
                The science behind it
              </p>
              <h2 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">
                Adaptive mechanics, not buzzwords
              </h2>
              <p className="mt-4 text-lg leading-relaxed text-slate-400">
                Routing, misconceptions, and review scheduling live next to grading and content—because
                that's where they actually affect outcomes.
              </p>
            </div>

            <div className="mt-12 grid gap-5 lg:grid-cols-3">
              {[
                {
                  icon: BrainCircuit,
                  title: 'Item Response Theory engine',
                  body: "Questions are calibrated using IRT 2PL/3PL models. The system estimates each learner's mastery level in real time and routes them to the content they actually need next—not what's next in the syllabus.",
                  color: 'text-indigo-400',
                  bg: 'bg-indigo-500/10 ring-indigo-500/20',
                },
                {
                  icon: GraduationCap,
                  title: 'Misconception detection',
                  body: 'AI analyzes incorrect responses across the class and surfaces the most common errors to instructors—so they can address patterns in the next session instead of marking them wrong and moving on.',
                  color: 'text-violet-400',
                  bg: 'bg-violet-500/10 ring-violet-500/20',
                },
                {
                  icon: RefreshCw,
                  title: 'Spaced repetition scheduler',
                  body: 'The SRS engine schedules review material at scientifically optimal intervals. Knowledge sticks between sessions instead of fading before the final exam.',
                  color: 'text-emerald-400',
                  bg: 'bg-emerald-500/10 ring-emerald-500/20',
                },
              ].map(({ icon: Icon, title, body, color, bg }) => (
                <div key={title} className="feature-card-dark">
                  <div className={`flex h-11 w-11 items-center justify-center rounded-xl ring-1 ${bg} ${color}`}>
                    <Icon className="h-5 w-5" aria-hidden />
                  </div>
                  <h3 className="mt-5 text-base font-semibold text-white">{title}</h3>
                  <p className="mt-2.5 text-sm leading-relaxed text-slate-400">{body}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ══════════════════════════════════ INSTITUTIONS ═══ */}
        <section id="institutions" className="bg-surface py-20 sm:py-28">
          <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <div className="grid gap-12 lg:grid-cols-2 lg:items-center lg:gap-20">
              <div>
                <p className="text-[0.7rem] font-semibold uppercase tracking-[0.22em] text-indigo-500">
                  For institutions
                </p>
                <h2 className="mt-3 text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl">
                  Built around how institutions actually operate
                </h2>
                <p className="mt-5 text-lg leading-relaxed text-slate-600">
                  From K-12 districts to university programs, Lextures is modeled on the workflows
                  that break first at scale: enrollment drift, inconsistent accommodations, grading
                  that can't be audited, and content no one can keep synchronized.
                </p>
                <ul className="mt-8 space-y-4">
                  {[
                    'Course blueprints let coordinators push updates to every child section at once—syllabus changes, rubrics, new items.',
                    'Accommodations are configured once at the platform level and applied to every assessment automatically.',
                    'Every grading action is logged with who changed what, when, and why—a complete paper trail for appeals and accreditation.',
                  ].map((line) => (
                    <li key={line} className="flex gap-3 text-slate-700">
                      <span className="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-indigo-500" aria-hidden />
                      <span className="text-[0.9375rem] leading-relaxed">{line}</span>
                    </li>
                  ))}
                </ul>
                <div className="mt-9 flex flex-wrap gap-3">
                  <a href="/higher-ed" className="btn-primary gap-2">
                    Higher Education
                    <ArrowRight className="h-4 w-4" aria-hidden />
                  </a>
                  <a href="/k-12" className="btn-secondary">
                    K–12
                  </a>
                </div>
              </div>

              {/* Quote card */}
              <div className="relative rounded-3xl border border-slate-200 bg-white p-8 shadow-[0_4px_40px_rgba(15,23,42,0.06)]">
                <div
                  aria-hidden
                  className="pointer-events-none absolute -right-10 -top-10 h-40 w-40 rounded-full bg-indigo-100 blur-[50px]"
                />
                <p className="text-[0.7rem] font-semibold uppercase tracking-[0.2em] text-slate-400">
                  Design principle
                </p>
                <blockquote className="relative mt-5">
                  <span aria-hidden className="absolute -left-1 -top-3 text-6xl leading-none text-indigo-200 select-none font-display">"</span>
                  <p className="relative z-10 pl-5 text-xl font-medium leading-snug text-slate-900">
                    If a registrar would wince at the data model, it doesn't ship—operational
                    honesty beats feature checklists.
                  </p>
                </blockquote>
                <p className="mt-5 text-sm leading-relaxed text-slate-500">
                  Lextures is under active development. The public demo is the fastest way to see
                  current capabilities and where the product is heading.
                </p>
                <a href="/get-started" className="btn-primary mt-6 inline-flex gap-2">
                  Try the live demo
                  <ArrowRight className="h-4 w-4" aria-hidden />
                </a>
              </div>
            </div>
          </div>
        </section>

        {/* ════════════════════════════════ INTEGRATIONS ═══ */}
        <section id="integrations" className="bg-white py-20 sm:py-28">
          <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <div className="grid gap-12 lg:grid-cols-[1fr_1.2fr] lg:items-start">
              <div>
                <p className="text-[0.7rem] font-semibold uppercase tracking-[0.22em] text-indigo-500">
                  Integrations
                </p>
                <h2 className="mt-3 text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl">
                  Works inside the stack you already have
                </h2>
                <p className="mt-4 text-lg leading-relaxed text-slate-500">
                  No rip-and-replace. Lextures integrates with Canvas, Blackboard, Moodle, and
                  your district SIS—or stands alone as your primary LMS.
                </p>
                <div className="mt-8 flex flex-wrap gap-2">
                  {[
                    'LTI 1.3', 'SAML 2.0', 'OIDC', 'OneRoster 1.2',
                    'SCIM 2.0', 'Clever', 'ClassLink', 'QTI 2.1/3.0', 'Canvas import', 'iCalendar',
                  ].map(tag => (
                    <span
                      key={tag}
                      className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-semibold text-slate-600"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              </div>

              <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                {[
                  {
                    term: 'LTI 1.3 provider & consumer',
                    desc: 'Launch Lextures inside Canvas, Blackboard, or Moodle. Roster sync via NRPS, grade passback via AGS, and deep linking for publisher content—all spec-compliant.',
                  },
                  {
                    term: 'Canvas import',
                    desc: 'Move courses, question banks, and grades from Canvas using WebSocket-based import with AI-assisted migration. QTI 2.1/3.0 from any major LMS.',
                  },
                  {
                    term: 'SSO & roster provisioning',
                    desc: 'SAML 2.0, OIDC, Clever, and ClassLink for identity. OneRoster 1.2 CSV and SCIM 2.0 HTTP for roster sync. Auto-provision users on first login.',
                  },
                  {
                    term: 'Defensible exports',
                    desc: 'Grade exports structured for accreditation reviews and appeals. iCalendar feeds for due dates. QTI exports for question bank portability.',
                  },
                ].map(row => (
                  <div key={row.term} className="feature-card">
                    <dt className="text-sm font-semibold text-slate-900">{row.term}</dt>
                    <dd className="mt-2 text-sm leading-relaxed text-slate-500">{row.desc}</dd>
                  </div>
                ))}
              </dl>
            </div>
          </div>
        </section>

        {/* ═══════════════════════════════════ OPEN SOURCE ═══ */}
        <section className="relative overflow-hidden bg-slate-950 py-20 sm:py-24">
          <div
            aria-hidden
            className="pointer-events-none absolute left-0 top-0 h-full w-1/2 bg-gradient-to-r from-indigo-600/10 to-transparent"
          />
          <div className="relative mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <div className="flex flex-col gap-10 lg:flex-row lg:items-center lg:gap-16">
              <div className="lg:max-w-md">
                <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-400 ring-1 ring-indigo-500/20">
                  <Code2 className="h-5 w-5" aria-hidden />
                </div>
                <h2 className="mt-5 text-2xl font-bold tracking-tight text-white sm:text-3xl">
                  Open source, AGPL-3.0 licensed
                </h2>
                <p className="mt-4 leading-relaxed text-slate-400">
                  The full stack—Go backend, React frontend, database migrations—is public on
                  GitHub. Deploy on your own infrastructure, fork it, or contribute. No vendor
                  lock-in, no usage fees.
                </p>
                <div className="mt-6 flex gap-3">
                  <a href={LINKS.github} className="btn-primary gap-2">
                    View on GitHub
                    <ArrowRight className="h-4 w-4" aria-hidden />
                  </a>
                </div>
              </div>

              {/* Terminal */}
              <div className="flex-1 overflow-hidden rounded-2xl border border-slate-700/60 bg-[#0d1117] shadow-[0_20px_60px_rgba(0,0,0,0.5)]">
                <div className="flex items-center gap-1.5 border-b border-slate-700/60 px-4 py-3">
                  <span className="h-3 w-3 rounded-full bg-red-500/70" aria-hidden />
                  <span className="h-3 w-3 rounded-full bg-yellow-500/70" aria-hidden />
                  <span className="h-3 w-3 rounded-full bg-green-500/70" aria-hidden />
                  <span className="ml-3 text-[0.68rem] font-medium text-slate-500">zsh — lextures</span>
                </div>
                <div className="p-5 font-mono text-sm leading-[1.8]">
                  <p className="text-slate-500"># Get started in minutes</p>
                  <p className="mt-2">
                    <span className="text-emerald-400">$</span>{' '}
                    <span className="text-indigo-300">git clone</span>{' '}
                    <span className="text-slate-300">https://github.com/StudyDrift/lextures</span>
                  </p>
                  <p>
                    <span className="text-emerald-400">$</span>{' '}
                    <span className="text-indigo-300">cd</span>{' '}
                    <span className="text-slate-300">lextures</span>
                  </p>
                  <p className="mt-2 text-slate-500"># First admin (before signup)</p>
                  <p>
                    <span className="text-emerald-400">$</span>{' '}
                    <span className="text-indigo-300">echo</span>{' '}
                    <span className="text-amber-200/90">&apos;BOOTSTRAP_ADMIN_EMAIL=you@example.com&apos;</span>{' '}
                    <span className="text-indigo-300">&gt;&gt; .env</span>
                  </p>
                  <p className="mt-2 text-slate-500"># Dev stack (web :5173, API :8080)</p>
                  <p>
                    <span className="text-emerald-400">$</span>{' '}
                    <span className="text-indigo-300">docker compose</span>{' '}
                    <span className="text-slate-300">-f docker-compose.yml -f docker-compose.dev.yml up --build -d</span>
                  </p>
                  <p className="mt-2">
                    <span className="text-slate-500">✓</span>{' '}
                    <span className="text-emerald-400">Sign up at</span>{' '}
                    <span className="text-indigo-400 underline underline-offset-2">http://localhost:5173</span>
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* ══════════════════════════════════════ FINAL CTA ═══ */}
        <section className="relative overflow-hidden bg-gradient-to-br from-indigo-600 via-indigo-600 to-violet-700 py-24 sm:py-32">
          {/* Subtle dot pattern */}
          <div
            aria-hidden
            className="pointer-events-none absolute inset-0 opacity-30"
            style={{
              backgroundImage: 'radial-gradient(circle, rgba(255,255,255,0.15) 1px, transparent 1px)',
              backgroundSize: '24px 24px',
            }}
          />
          <div
            aria-hidden
            className="pointer-events-none absolute -left-20 top-0 h-80 w-80 rounded-full bg-white/10 blur-[80px]"
          />
          <div
            aria-hidden
            className="pointer-events-none absolute -right-20 bottom-0 h-80 w-80 rounded-full bg-violet-400/20 blur-[80px]"
          />

          <div className="relative mx-auto max-w-4xl px-4 text-center sm:px-6 lg:px-8">
            <p className="text-[0.7rem] font-semibold uppercase tracking-[0.22em] text-indigo-200">
              Ready to explore?
            </p>
            <h2 className="font-display mt-5 text-4xl font-normal italic leading-tight tracking-tight text-white sm:text-5xl">
              Try the product on the hosted demo
            </h2>
            <p className="mx-auto mt-5 max-w-xl text-lg leading-relaxed text-indigo-200">
              Walk learner and instructor flows—quizzes, gradebook, imports—then decide if the
              stack matches your institution. No login required.
            </p>
            <div className="mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row">
              <a href="/get-started" className="inline-flex h-12 cursor-pointer items-center justify-center gap-2 rounded-xl bg-white px-8 text-[0.9375rem] font-semibold text-indigo-700 shadow-sm transition-all duration-150 hover:bg-indigo-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white">
                Get Started free
                <ArrowRight className="h-4 w-4" aria-hidden />
              </a>
              <a href={LINKS.github} className="btn-ghost-dark h-12 px-8 text-[0.9375rem]">
                Study the source
              </a>
            </div>
          </div>
        </section>
      </main>

      <SiteFooter />
    </div>
  )
}

/* ─────────────────────────────────────────────────────────
   ROUTER
   ───────────────────────────────────────────────────────── */
function resolveRoute(pathname: string, hash: string): string {
  if (hash.startsWith('#/')) {
    const hashRoute = hash.slice(1)
    // e.g. /blog + #/blog/my-post — pathname wins today and breaks post links
    if (pathname !== '/' && hashRoute.startsWith(`${pathname}/`)) {
      return hashRoute
    }
    if (pathname === '/') return hashRoute
  }
  return pathname !== '/' ? pathname : '/'
}

export default function App() {
  const hash = useHashRoute()
  const route = resolveRoute(window.location.pathname, hash)

  if (route === '/get-started') return <GetStartedPage />
  if (route === '/higher-ed') return <HigherEdPage />
  if (route === '/k-12') return <K12Page />
  if (route === '/self-learner') return <SelfLearnerPage />
  if (route === '/pricing') return <PricingPage />
  if (route === '/blog') return <BlogIndex />
  if (route.startsWith('/blog/')) return <BlogPost slug={route.slice('/blog/'.length)} />
  if (route === '/docs') return <DocsIndex />
  if (route.startsWith('/docs/')) return <DocsPost slug={route.slice('/docs/'.length)} />
  if (route === '/privacy') return <PrivacyPolicyPage />
  if (route === '/privacy/history') return <PrivacyPolicyHistoryPage />
  if (route === '/terms') return <TermsOfServicePage />
  if (route === '/terms/history') return <TermsOfServiceHistoryPage />
  if (route === '/security') return <SecurityPage />
  if (route === '/accessibility') return <AccessibilityConformancePage />
  if (route === '/accessibility/vpat') return <VpatPage />
  if (route === '/privacy-rights/california') return <CaliforniaPrivacyRightsPage />
  return <HomePage />
}
