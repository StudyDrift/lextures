import { SITE_LINKS } from './site-links'

export type InstitutionInquiryForm = {
  organizationType: string
  organizationName: string
  contactName: string
  email: string
  role: string
  enrollmentSize: string
  hostingPreference: string
  message: string
}

function line(label: string, value: string): string {
  const trimmed = value.trim()
  return `${label}: ${trimmed || '—'}`
}

export function buildInstitutionInquiryMailto(form: InstitutionInquiryForm): string {
  const subject = `Lextures institution inquiry — ${form.organizationName.trim()}`
  const body = [
    line('Organization type', form.organizationType),
    line('Organization', form.organizationName),
    line('Contact name', form.contactName),
    line('Email', form.email),
    line('Role / title', form.role),
    line('Enrollment size', form.enrollmentSize),
    line('Hosting preference', form.hostingPreference),
    '',
    'Message:',
    form.message.trim(),
  ].join('\n')

  const params = new URLSearchParams()
  params.set('subject', subject)
  params.set('body', body)
  if (form.email.trim()) {
    params.set('cc', form.email.trim())
  }

  return `mailto:${SITE_LINKS.institutionInquiryEmail}?${params.toString()}`
}
