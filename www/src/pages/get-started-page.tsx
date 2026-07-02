import { ArrowLeft, BrainCircuit, GraduationCap, KeyRound } from 'lucide-react'
import { useState } from 'react'
import { Header } from '../components/header'
import { SiteFooter } from '../components/site-footer'
import { isValidSchoolCode, normalizeSchoolCode, schoolCodeError } from '../lib/school-code'
import { SITE_LINKS, TENANT_HOST_SUFFIX, tenantOrigin } from '../lib/site-links'

// Fire-and-forget — never awaited, never surfaces errors to the user.
function trackOnboarding(program: string, schoolCode?: string) {
  try {
    const apiBase = import.meta.env.VITE_API_BASE_URL ?? ''
    navigator.sendBeacon(
      `${apiBase}/api/v1/public/onboarding/track`,
      new Blob(
        [JSON.stringify({
          program,
          school_name: schoolCode ?? '',
          language: navigator.language ?? '',
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone ?? '',
          screen_width: window.screen.width,
          screen_height: window.screen.height,
          referrer: document.referrer,
        })],
        { type: 'application/json' },
      ),
    )
  } catch {
    // Never let analytics break the user flow.
  }
}

type Path = 'self-learner' | 'school'
type Step = 'choose' | 'school-code'

const PATHS = [
  {
    id: 'self-learner' as Path,
    icon: BrainCircuit,
    title: 'Self-learner',
    description: "I'm studying independently, for a certification, or on my own schedule.",
  },
  {
    id: 'school' as Path,
    icon: GraduationCap,
    title: 'School',
    description: "I'm a student or educator at a school that uses Lextures.",
  },
]

const fieldClass =
  'block w-full rounded-xl border border-slate-200 bg-white py-3 text-base text-slate-900 shadow-sm outline-none transition-colors focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20'

function BackButton({ onClick }: { onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="mb-8 flex items-center gap-1.5 text-sm font-medium text-slate-500 transition-colors hover:text-slate-900"
    >
      <ArrowLeft className="h-3.5 w-3.5" aria-hidden />
      Back
    </button>
  )
}

function ChooseStep({ onSelect }: { onSelect: (path: Path) => void }) {
  return (
    <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6 sm:py-24 lg:px-8">
      <div className="text-center">
        <h1 className="text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">
          How are you using Lextures?
        </h1>
        <p className="mt-3 text-base leading-relaxed text-slate-500">
          Choose the option that matches how you sign in.
        </p>
      </div>

      <div className="mx-auto mt-12 grid max-w-2xl gap-4 sm:grid-cols-2">
        {PATHS.map(({ id, icon: Icon, title, description }) => (
          <button
            key={id}
            type="button"
            onClick={() => onSelect(id)}
            className="group flex cursor-pointer flex-col items-start gap-4 rounded-2xl border border-slate-200 bg-white p-6 text-left shadow-[0_1px_3px_rgba(28,25,23,0.05)] transition-all duration-150 hover:border-accent hover:shadow-[0_4px_16px_rgba(15,118,110,0.12)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500"
          >
            <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-indigo-50 text-indigo-600 ring-1 ring-indigo-200 transition-colors group-hover:bg-accent group-hover:text-white">
              <Icon className="h-5 w-5" aria-hidden />
            </div>
            <div>
              <p className="font-semibold text-slate-900">{title}</p>
              <p className="mt-1.5 text-sm leading-relaxed text-slate-500">{description}</p>
            </div>
          </button>
        ))}
      </div>
    </div>
  )
}

function SchoolCodeStep({ onBack }: { onBack: () => void }) {
  const [code, setCode] = useState('')
  const normalizedCode = normalizeSchoolCode(code)
  const error = code ? schoolCodeError(code) : null
  const previewHost = normalizedCode ? `${normalizedCode}.${TENANT_HOST_SUFFIX}` : `your-school.${TENANT_HOST_SUFFIX}`

  function handleContinue() {
    if (!isValidSchoolCode(code)) return
    const schoolCode = normalizeSchoolCode(code)
    trackOnboarding('school', schoolCode)
    window.location.href = tenantOrigin(schoolCode)
  }

  return (
    <div className="mx-auto max-w-lg px-4 py-16 sm:px-6 sm:py-24 lg:px-8">
      <BackButton onClick={onBack} />

      <h1 className="text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">
        Enter your school code
      </h1>
      <p className="mt-3 text-base leading-relaxed text-slate-500">
        Your school or district provides a short code for sign-in. Enter it below and we&apos;ll take
        you to your school&apos;s Lextures site.
      </p>

      <div className="mt-10 space-y-4">
        <div>
          <label htmlFor="school-code" className="sr-only">
            School code
          </label>
          <div className="relative">
            <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
              <KeyRound className="h-4 w-4 text-slate-400" aria-hidden />
            </div>
            <input
              id="school-code"
              type="text"
              autoFocus
              autoComplete="organization"
              spellCheck={false}
              value={code}
              onChange={e => setCode(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleContinue()}
              placeholder="e.g. example"
              className={`${fieldClass} pl-10 pr-4 placeholder-stone-400`}
              aria-invalid={error ? true : undefined}
              aria-describedby="school-code-help school-code-preview"
            />
          </div>
          <p id="school-code-help" className="mt-2 text-sm text-slate-500">
            Example: <span className="font-medium text-slate-700">example</span> opens{' '}
            <span className="font-medium text-slate-700">example.lextures.com</span>.
          </p>
          {error && (
            <p className="mt-2 text-sm text-red-600" role="alert">
              {error}
            </p>
          )}
        </div>

        <div
          id="school-code-preview"
          className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600"
        >
          You&apos;ll go to{' '}
          <span className="font-medium text-slate-900">{previewHost}</span>
        </div>

        <button
          type="button"
          onClick={handleContinue}
          disabled={!isValidSchoolCode(code)}
          className="btn-primary w-full justify-center py-3 text-base disabled:cursor-not-allowed disabled:opacity-40"
        >
          Continue
        </button>
      </div>
    </div>
  )
}

export function GetStartedPage() {
  const [step, setStep] = useState<Step>('choose')

  function handleChoose(path: Path) {
    if (path === 'self-learner') {
      trackOnboarding('self-learner')
      window.location.href = SITE_LINKS.selfLearner
      return
    }
    setStep('school-code')
  }

  return (
    <div className="relative min-h-screen overflow-x-hidden bg-white text-slate-900">
      <Header />

      <main className="flex min-h-[calc(100vh-4rem)] items-start justify-center">
        {step === 'choose' && <ChooseStep onSelect={handleChoose} />}
        {step === 'school-code' && <SchoolCodeStep onBack={() => setStep('choose')} />}
      </main>

      <SiteFooter />
    </div>
  )
}