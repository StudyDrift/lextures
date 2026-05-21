import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { LegalDocumentPage } from '../legal-document-page'
import { PRIVACY_POLICY } from '../../../lib/legal-documents'

describe('LegalDocumentPage', () => {
  it('renders title, version, and FERPA section without authentication', () => {
    render(
      <MemoryRouter>
        <LegalDocumentPage document={PRIVACY_POLICY} />
      </MemoryRouter>,
    )
    expect(screen.getByRole('heading', { level: 1, name: /privacy policy/i })).toBeInTheDocument()
    expect(screen.getByText(new RegExp(PRIVACY_POLICY.effectiveDateLabel))).toBeInTheDocument()
    expect(screen.getByText(PRIVACY_POLICY.version)).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /ferpa and student education records/i })).toBeInTheDocument()
    expect(screen.getByRole('navigation', { name: /table of contents/i })).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: /history of changes/i }).length).toBeGreaterThan(0)
  })

  it('includes GDPR rights section for accessibility navigation', () => {
    render(
      <MemoryRouter>
        <LegalDocumentPage document={PRIVACY_POLICY} />
      </MemoryRouter>,
    )
    expect(screen.getByRole('heading', { name: /your rights under gdpr/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /jump to rights/i })).toHaveAttribute('href', '#your-rights-under-gdpr')
  })
})
