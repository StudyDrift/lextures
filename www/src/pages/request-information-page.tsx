import { ArrowLeft } from 'lucide-react'
import { useEffect, useState, type FormEvent } from 'react'
import { MarketingPageShell } from '../components/marketing-page-shell'
import {
  InstitutionInquiryApiError,
  submitInstitutionInquiry,
} from '../lib/institution-inquiry-api'
import type { InstitutionInquiryForm } from '../lib/institution-inquiry-mailto'
import { SITE_LINKS } from '../lib/site-links'

const INITIAL_FORM: InstitutionInquiryForm = {
  organizationType: 'University',
  organizationName: '',
  contactName: '',
  email: '',
  role: '',
  enrollmentSize: '',
  hostingPreference: 'Not sure yet',
  message: '',
}

const fieldClass =
  'block w-full rounded-lg border px-3.5 py-2.5 text-[15px] outline-none transition-colors focus-visible:ring-2'

const fieldStyle = {
  backgroundColor: 'var(--panel)',
  borderColor: 'var(--line-card)',
  color: 'var(--ink-nav)',
} as const

function FieldLabel({ htmlFor, children }: { htmlFor: string; children: string }) {
  return (
    <label
      htmlFor={htmlFor}
      className="mb-1.5 block text-[14px] font-medium"
      style={{ color: 'var(--ink-nav)' }}
    >
      {children}
    </label>
  )
}

export function RequestInformationPage() {
  const [form, setForm] = useState<InstitutionInquiryForm>(INITIAL_FORM)
  const [submitted, setSubmitted] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    document.title = 'Request information — Lextures'
  }, [])

  function updateField<K extends keyof InstitutionInquiryForm>(key: K, value: InstitutionInquiryForm[K]) {
    setForm(current => ({ ...current, [key]: value }))
    setError(null)
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (submitting) return
    setError(null)
    setSubmitting(true)
    try {
      await submitInstitutionInquiry(form)
      setSubmitted(true)
    } catch (err) {
      if (err instanceof InstitutionInquiryApiError) {
        setError(err.message)
      } else {
        setError("We couldn't submit your request right now. Check your connection and try again.")
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <MarketingPageShell>
      <section className="border-b py-16 md:py-20" style={{ borderColor: 'var(--line)' }}>
        <div className="mx-auto max-w-[640px] px-5 md:px-10 xl:px-14">
          <a
            href="/pricing"
            className="inline-flex items-center gap-1.5 text-[14px] font-medium no-underline"
            style={{ color: 'var(--text-soft)' }}
          >
            <ArrowLeft className="h-4 w-4" aria-hidden />
            Back to pricing
          </a>

          <p className="eyebrow-label mt-8">University or district</p>
          <h1
            className="font-display mt-4 text-[clamp(32px,4vw,40px)] font-semibold leading-tight tracking-[-0.02em]"
            style={{ color: 'var(--ink)' }}
          >
            Request information
          </h1>
          <p className="mt-4 text-[16px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
            Tell us about your institution and what you are evaluating. We&apos;ll follow up at the
            work email you provide. Prefer email? Reach us at{' '}
            <a href={`mailto:${SITE_LINKS.institutionInquiryEmail}`} className="underline underline-offset-2">
              {SITE_LINKS.institutionInquiryEmail}
            </a>
            .
          </p>

          {submitted ? (
            <div
              className="mt-10 border p-6"
              style={{
                backgroundColor: 'var(--teal-tint)',
                borderColor: 'var(--teal)',
                borderRadius: 'var(--radius-card)',
              }}
              role="status"
            >
              <p className="text-[15px] font-semibold" style={{ color: 'var(--ink-nav)' }}>
                Thanks — we received your request
              </p>
              <p className="mt-2 text-[14px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
                Our team will review the details and get back to you. If you need to add anything,
                email{' '}
                <a href={`mailto:${SITE_LINKS.institutionInquiryEmail}`} className="underline underline-offset-2">
                  {SITE_LINKS.institutionInquiryEmail}
                </a>
                .
              </p>
            </div>
          ) : (
            <form className="mt-10 space-y-5" onSubmit={e => void handleSubmit(e)}>
              <div>
                <FieldLabel htmlFor="organizationType">Organization type</FieldLabel>
                <select
                  id="organizationType"
                  required
                  value={form.organizationType}
                  onChange={event => updateField('organizationType', event.target.value)}
                  disabled={submitting}
                  className={fieldClass}
                  style={fieldStyle}
                >
                  <option value="University">University or college</option>
                  <option value="K-12 district">K–12 district</option>
                  <option value="Multi-campus system">Multi-campus system</option>
                  <option value="Other">Other institution</option>
                </select>
              </div>

              <div>
                <FieldLabel htmlFor="organizationName">Organization name</FieldLabel>
                <input
                  id="organizationName"
                  type="text"
                  required
                  autoComplete="organization"
                  value={form.organizationName}
                  onChange={event => updateField('organizationName', event.target.value)}
                  placeholder="e.g. State University or Lincoln Unified School District"
                  disabled={submitting}
                  className={fieldClass}
                  style={fieldStyle}
                />
              </div>

              <div className="grid gap-5 sm:grid-cols-2">
                <div>
                  <FieldLabel htmlFor="contactName">Your name</FieldLabel>
                  <input
                    id="contactName"
                    type="text"
                    required
                    autoComplete="name"
                    value={form.contactName}
                    onChange={event => updateField('contactName', event.target.value)}
                    disabled={submitting}
                    className={fieldClass}
                    style={fieldStyle}
                  />
                </div>
                <div>
                  <FieldLabel htmlFor="email">Work email</FieldLabel>
                  <input
                    id="email"
                    type="email"
                    required
                    autoComplete="email"
                    value={form.email}
                    onChange={event => updateField('email', event.target.value)}
                    disabled={submitting}
                    className={fieldClass}
                    style={fieldStyle}
                  />
                </div>
              </div>

              <div>
                <FieldLabel htmlFor="role">Role or title</FieldLabel>
                <input
                  id="role"
                  type="text"
                  autoComplete="organization-title"
                  value={form.role}
                  onChange={event => updateField('role', event.target.value)}
                  placeholder="e.g. Director of IT, Registrar, Superintendent"
                  disabled={submitting}
                  className={fieldClass}
                  style={fieldStyle}
                />
              </div>

              <div className="grid gap-5 sm:grid-cols-2">
                <div>
                  <FieldLabel htmlFor="enrollmentSize">Approximate enrollment</FieldLabel>
                  <select
                    id="enrollmentSize"
                    required
                    value={form.enrollmentSize}
                    onChange={event => updateField('enrollmentSize', event.target.value)}
                    disabled={submitting}
                    className={fieldClass}
                    style={fieldStyle}
                  >
                    <option value="" disabled>
                      Select a range
                    </option>
                    <option value="Under 1,000">Under 1,000</option>
                    <option value="1,000 – 10,000">1,000 – 10,000</option>
                    <option value="10,000 – 50,000">10,000 – 50,000</option>
                    <option value="Over 50,000">Over 50,000</option>
                    <option value="Not sure">Not sure</option>
                  </select>
                </div>
                <div>
                  <FieldLabel htmlFor="hostingPreference">Hosting preference</FieldLabel>
                  <select
                    id="hostingPreference"
                    required
                    value={form.hostingPreference}
                    onChange={event => updateField('hostingPreference', event.target.value)}
                    disabled={submitting}
                    className={fieldClass}
                    style={fieldStyle}
                  >
                    <option value="Not sure yet">Not sure yet</option>
                    <option value="Self-host on our infrastructure">Self-host on our infrastructure</option>
                    <option value="Hosted by Lextures">Hosted by Lextures</option>
                    <option value="Managed deployment on our cloud">Managed deployment on our cloud</option>
                  </select>
                </div>
              </div>

              <div>
                <FieldLabel htmlFor="message">What are you looking for?</FieldLabel>
                <textarea
                  id="message"
                  required
                  rows={5}
                  value={form.message}
                  onChange={event => updateField('message', event.target.value)}
                  placeholder="Timeline, SSO requirements, LMS integration, pilot scope, support needs…"
                  disabled={submitting}
                  className={`${fieldClass} resize-y`}
                  style={fieldStyle}
                />
              </div>

              {error && (
                <p className="text-sm text-red-600" role="alert">
                  {error}
                </p>
              )}

              <button
                type="submit"
                disabled={submitting}
                aria-busy={submitting || undefined}
                className="btn-primary w-full justify-center sm:w-auto disabled:cursor-not-allowed disabled:opacity-60"
              >
                {submitting ? 'Submitting…' : 'Submit request'}
              </button>
            </form>
          )}
        </div>
      </section>
    </MarketingPageShell>
  )
}
