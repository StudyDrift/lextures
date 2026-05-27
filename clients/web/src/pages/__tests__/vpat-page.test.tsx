import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import VpatPage from '../vpat-page'
import {
  WCAG_CRITERIA,
  FPC_CRITERIA,
  SEC508_SOFTWARE_CRITERIA,
  SEC508_SUPPORT_CRITERIA,
  EN301549_SOFTWARE_CRITERIA,
  EN301549_SUPPORT_CRITERIA,
} from '../../lib/vpat-data'

function renderPage() {
  return render(
    <MemoryRouter>
      <VpatPage />
    </MemoryRouter>,
  )
}

describe('VpatPage', () => {
  beforeEach(() => {
    vi.spyOn(document, 'title', 'set')
  })

  it('sets the document title', () => {
    renderPage()
    expect(document.title).toContain('VPAT')
    expect(document.title).toContain('Lextures')
  })

  it('renders the main heading', () => {
    renderPage()
    expect(screen.getByRole('heading', { level: 1, name: /accessibility conformance report/i })).toBeInTheDocument()
  })

  it('shows the VPAT version badge', () => {
    renderPage()
    expect(screen.getByText(/VPAT.*2\.5.*INT/i)).toBeInTheDocument()
  })

  it('renders product information section', () => {
    renderPage()
    expect(screen.getByRole('heading', { name: /product information/i })).toBeInTheDocument()
    expect(screen.getByText('Lextures')).toBeInTheDocument()
    // Multiple elements contain this date; check at least one is present
    expect(screen.getAllByText(/May 27, 2026/).length).toBeGreaterThanOrEqual(1)
  })

  it('renders the download section with a download link', () => {
    renderPage()
    const downloadLink = screen.getByRole('link', { name: /download vpat/i })
    expect(downloadLink).toBeInTheDocument()
    expect(downloadLink).toHaveAttribute('download')
    expect(downloadLink).toHaveAttribute('href', expect.stringMatching(/VPAT_2\.5_INT_Lextures/))
  })

  it('renders table of contents navigation', () => {
    renderPage()
    const toc = screen.getByRole('navigation', { name: /report sections/i })
    expect(toc).toBeInTheDocument()
    expect(toc.querySelectorAll('li').length).toBeGreaterThan(5)
  })

  describe('WCAG 2.1 tables', () => {
    it('renders Level A section heading', () => {
      renderPage()
      expect(screen.getByRole('heading', { name: /level a success criteria/i })).toBeInTheDocument()
    })

    it('renders Level AA section heading', () => {
      renderPage()
      expect(screen.getByRole('heading', { name: /level aa success criteria/i })).toBeInTheDocument()
    })

    it('renders all WCAG criteria rows', () => {
      renderPage()
      const allScs = WCAG_CRITERIA.map((c) => c.sc)
      for (const sc of allScs) {
        expect(screen.getByText(sc)).toBeInTheDocument()
      }
    })

    it('renders all Level A criteria (at least 29)', () => {
      renderPage()
      const levelA = WCAG_CRITERIA.filter((c) => c.level === 'A')
      expect(levelA.length).toBeGreaterThanOrEqual(29)
    })

    it('renders all Level AA criteria (at least 19)', () => {
      renderPage()
      const levelAA = WCAG_CRITERIA.filter((c) => c.level === 'AA')
      expect(levelAA.length).toBeGreaterThanOrEqual(19)
    })

    it('shows 1.2.2 Captions as Partially Supports', () => {
      renderPage()
      const captions = WCAG_CRITERIA.find((c) => c.sc === '1.2.2')
      expect(captions?.conformance).toBe('Partially Supports')
    })
  })

  describe('Section 508 tables', () => {
    it('renders Functional Performance Criteria section', () => {
      renderPage()
      expect(screen.getByRole('heading', { name: /chapter 3.*functional performance/i })).toBeInTheDocument()
    })

    it('renders all FPC criteria rows', () => {
      renderPage()
      for (const c of FPC_CRITERIA) {
        expect(screen.getByText(c.id)).toBeInTheDocument()
      }
    })

    it('renders Chapter 5 Software section', () => {
      renderPage()
      expect(screen.getByRole('heading', { name: /chapter 5.*software/i })).toBeInTheDocument()
    })

    it('renders all Chapter 5 criteria rows', () => {
      renderPage()
      for (const c of SEC508_SOFTWARE_CRITERIA) {
        expect(screen.getByText(c.id)).toBeInTheDocument()
      }
    })

    it('renders Chapter 6 Support Documentation section', () => {
      renderPage()
      expect(screen.getByRole('heading', { name: /chapter 6.*support documentation/i })).toBeInTheDocument()
    })

    it('renders all Chapter 6 criteria rows', () => {
      renderPage()
      for (const c of SEC508_SUPPORT_CRITERIA) {
        expect(screen.getByText(c.id)).toBeInTheDocument()
      }
    })
  })

  describe('EN 301 549 tables', () => {
    it('renders EN 301 549 Chapter 9 section', () => {
      renderPage()
      expect(screen.getByRole('heading', { name: /chapter 9.*web/i })).toBeInTheDocument()
    })

    it('renders EN 301 549 Chapter 11 section', () => {
      renderPage()
      expect(screen.getByRole('heading', { name: /chapter 11.*software/i })).toBeInTheDocument()
    })

    it('renders all Chapter 11 criteria rows', () => {
      renderPage()
      for (const c of EN301549_SOFTWARE_CRITERIA) {
        expect(screen.getByText(c.clause)).toBeInTheDocument()
      }
    })

    it('renders EN 301 549 Chapter 12 section', () => {
      renderPage()
      expect(screen.getByRole('heading', { name: /chapter 12.*documentation/i })).toBeInTheDocument()
    })

    it('renders all Chapter 12 criteria rows', () => {
      renderPage()
      for (const c of EN301549_SUPPORT_CRITERIA) {
        expect(screen.getByText(c.clause)).toBeInTheDocument()
      }
    })
  })

  describe('Accessibility support contact', () => {
    it('renders Contact Accessibility Support section', () => {
      renderPage()
      expect(screen.getByRole('heading', { name: /contact accessibility support/i })).toBeInTheDocument()
    })

    it('has a contact link with pre-populated accommodation subject', () => {
      renderPage()
      // Multiple links with this name exist (TOC anchor + mailto); find the mailto one
      const contactLinks = screen.getAllByRole('link', { name: /contact accessibility support/i })
      const mailtoLink = contactLinks.find((l) =>
        l.getAttribute('href')?.startsWith('mailto:'),
      )
      expect(mailtoLink).toBeTruthy()
      expect(mailtoLink).toHaveAttribute(
        'href',
        'mailto:accessibility@lextures.com?subject=Accessibility%20accommodation%20request',
      )
    })
  })

  describe('Legal disclaimer', () => {
    it('renders the legal disclaimer section', () => {
      renderPage()
      expect(screen.getByRole('heading', { name: /legal disclaimer/i })).toBeInTheDocument()
      expect(screen.getByText(/VPAT.*registered trademark/i)).toBeInTheDocument()
    })
  })

  describe('Navigation', () => {
    it('has a skip to main content link', () => {
      renderPage()
      const skipLink = screen.getByRole('link', { name: /skip to main content/i })
      expect(skipLink).toHaveAttribute('href', '#main-content')
    })

    it('header nav contains Conformance Statement and legal links', () => {
      renderPage()
      const nav = screen.getByRole('navigation', { name: /legal/i })
      expect(nav.querySelector('a[href="/accessibility"]')).toBeTruthy()
      expect(nav.querySelector('a[href="/privacy"]')).toBeTruthy()
      expect(nav.querySelector('a[href="/terms"]')).toBeTruthy()
    })

    it('footer contains Accessibility Statement link', () => {
      renderPage()
      const footer = screen.getByRole('contentinfo')
      expect(footer.querySelector('a[href="/accessibility"]')).toBeTruthy()
    })
  })
})
