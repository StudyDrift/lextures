import { ArrowLeft, GraduationCap, House, KeyRound } from 'lucide-react'
import { useState } from 'react'
import { WindLines } from '../components/home/wind-lines'
import { MarketingPageShell } from '../components/marketing-page-shell'
import {
  isValidSchoolCode,
  lookupSchoolCode,
  normalizeSchoolCode,
  schoolCodeError,
  SCHOOL_LOOKUP_UNREACHABLE_MESSAGE,
  SCHOOL_NOT_FOUND_MESSAGE,
} from '../lib/school-code'
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

type Path = 'homeschool' | 'school'
type Step = 'choose' | 'school-code'

// Homeschool marketing segment for /get-started beacons (HS.2 FR-11 / HS.5).
// The API still accepts the pre-rebrand program value during the dual-read window.
const ONBOARDING_PROGRAM_HOMESCHOOL = 'homeschool'

const PATHS = [
  {
    id: 'homeschool' as Path,
    icon: House,
    title: 'Homeschool',
    description: "I'm homeschooling, studying for a certification, or learning on my own schedule.",
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
  const [lookupError, setLookupError] = useState<string | null>(null)
  const [checking, setChecking] = useState(false)
  const normalizedCode = normalizeSchoolCode(code)
  const formatError = code ? schoolCodeError(code) : null
  const error = formatError ?? lookupError
  const previewHost = normalizedCode ? `${normalizedCode}.${TENANT_HOST_SUFFIX}` : `your-school.${TENANT_HOST_SUFFIX}`
  const canContinue = isValidSchoolCode(code) && !checking

  async function handleContinue() {
    if (!isValidSchoolCode(code) || checking) return
    const schoolCode = normalizeSchoolCode(code)
    setLookupError(null)
    setChecking(true)
    try {
      const result = await lookupSchoolCode(schoolCode)
      if (!result.ok) {
        setLookupError(
          result.reason === 'not_found'
            ? SCHOOL_NOT_FOUND_MESSAGE
            : result.reason === 'invalid'
              ? (schoolCodeError(schoolCode) ?? SCHOOL_NOT_FOUND_MESSAGE)
              : SCHOOL_LOOKUP_UNREACHABLE_MESSAGE,
        )
        setChecking(false)
        return
      }
      trackOnboarding('school', result.slug)
      // Keep checking=true so the control stays disabled while the browser navigates.
      window.location.href = tenantOrigin(result.slug)
    } catch {
      setLookupError(SCHOOL_LOOKUP_UNREACHABLE_MESSAGE)
      setChecking(false)
    }
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
              onChange={e => {
                setCode(e.target.value)
                setLookupError(null)
              }}
              onKeyDown={e => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  void handleContinue()
                }
              }}
              disabled={checking}
              placeholder="e.g. example"
              className={`${fieldClass} pl-10 pr-4 placeholder-stone-400 disabled:opacity-60`}
              aria-invalid={error ? true : undefined}
              aria-describedby="school-code-help school-code-preview"
            />
          </div>
          <p id="school-code-help" className="mt-2 text-sm text-slate-500">
            Example: <span className="font-medium text-slate-700">example</span> opens{' '}
            <span className="font-medium text-slate-700">example.lextures.com</span>.
          </p>
          {error && (
            <div className="mt-2 space-y-1.5" role="alert">
              <p className="text-sm text-red-600">{error}</p>
              {lookupError === SCHOOL_NOT_FOUND_MESSAGE && !formatError && (
                <p className="text-sm text-slate-600">
                  If your school isn&apos;t using Lextures yet,{' '}
                  <a
                    href="/request-information"
                    className="font-medium text-indigo-600 underline-offset-2 hover:underline"
                  >
                    request that they adopt it
                  </a>
                  .
                </p>
              )}
            </div>
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
          onClick={() => void handleContinue()}
          disabled={!canContinue}
          aria-busy={checking || undefined}
          className="btn-primary w-full justify-center py-3 text-base disabled:cursor-not-allowed disabled:opacity-40"
        >
          {checking ? 'Checking…' : 'Continue'}
        </button>
      </div>
    </div>
  )
}

export function GetStartedPage() {
  const [step, setStep] = useState<Step>('choose')

  function handleChoose(path: Path) {
    if (path === 'homeschool') {
      trackOnboarding(ONBOARDING_PROGRAM_HOMESCHOOL)
      window.location.href = SITE_LINKS.homeschool
      return
    }
    setStep('school-code')
  }

  return (
    <MarketingPageShell>
      <section className="relative overflow-hidden">
        {/* Half the default hero wave opacity for a subtler ambient field on this page. */}
        <div className="pointer-events-none absolute inset-0" style={{ opacity: 0.5 }} aria-hidden>
          <WindLines variant="hero" />
        </div>
        <div className="relative z-[2] flex min-h-[calc(100vh-4rem)] items-start justify-center">
          {step === 'choose' && <ChooseStep onSelect={handleChoose} />}
          {step === 'school-code' && <SchoolCodeStep onBack={() => setStep('choose')} />}
        </div>
      </section>
    </MarketingPageShell>
  )
}