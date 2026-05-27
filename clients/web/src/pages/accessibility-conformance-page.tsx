import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { BrandLogo } from '../components/brand-logo'

type ConformanceLevel = 'Supports' | 'Partially Supports' | 'Does Not Support' | 'Not Applicable'

interface Criterion {
  sc: string
  title: string
  level: 'A' | 'AA'
  conformance: ConformanceLevel
  notes: string
}

const CRITERIA: Criterion[] = [
  // Level A
  { sc: '1.1.1', title: 'Non-text Content', level: 'A', conformance: 'Supports', notes: 'All UI images include alt text; the TipTap editor enforces alt text on uploaded images.' },
  { sc: '1.2.1', title: 'Audio-only and Video-only (Prerecorded)', level: 'A', conformance: 'Not Applicable', notes: 'Lextures does not host standalone audio-only or video-only content at this time.' },
  { sc: '1.2.2', title: 'Captions (Prerecorded)', level: 'A', conformance: 'Partially Supports', notes: 'Auto-captions for uploaded videos are in progress (plan 8.4).' },
  { sc: '1.2.3', title: 'Audio Description or Media Alternative (Prerecorded)', level: 'A', conformance: 'Not Applicable', notes: 'No prerecorded video content delivered by the platform itself.' },
  { sc: '1.3.1', title: 'Info and Relationships', level: 'A', conformance: 'Supports', notes: 'Semantic HTML headings, landmark regions, and table markup used throughout.' },
  { sc: '1.3.2', title: 'Meaningful Sequence', level: 'A', conformance: 'Supports', notes: 'DOM order matches visual reading order.' },
  { sc: '1.3.3', title: 'Sensory Characteristics', level: 'A', conformance: 'Supports', notes: 'Instructions do not rely solely on shape, size, or color.' },
  { sc: '1.4.1', title: 'Use of Color', level: 'A', conformance: 'Supports', notes: 'Color is never the sole means of conveying information; icons and text labels accompany color indicators.' },
  { sc: '1.4.2', title: 'Audio Control', level: 'A', conformance: 'Not Applicable', notes: 'No auto-playing audio.' },
  { sc: '2.1.1', title: 'Keyboard', level: 'A', conformance: 'Supports', notes: 'All interactive elements are keyboard accessible. Drag-and-drop module reorder provides a keyboard alternative (Space to grab, arrow keys to move, Enter to drop).' },
  { sc: '2.1.2', title: 'No Keyboard Trap', level: 'A', conformance: 'Supports', notes: 'Modal dialogs use a focus trap that releases on Escape or close button.' },
  { sc: '2.2.1', title: 'Timing Adjustable', level: 'A', conformance: 'Not Applicable', notes: 'No session timeouts or time limits on content.' },
  { sc: '2.2.2', title: 'Pause, Stop, Hide', level: 'A', conformance: 'Not Applicable', notes: 'No auto-moving, blinking, or scrolling content.' },
  { sc: '2.3.1', title: 'Three Flashes or Below Threshold', level: 'A', conformance: 'Supports', notes: 'No flashing content.' },
  { sc: '2.4.1', title: 'Bypass Blocks', level: 'A', conformance: 'Supports', notes: 'A "Skip to main content" link appears at the top of every authenticated page and becomes visible on focus.' },
  { sc: '2.4.2', title: 'Page Titled', level: 'A', conformance: 'Supports', notes: 'Each page updates document.title to include the page name and "Lextures".' },
  { sc: '2.4.3', title: 'Focus Order', level: 'A', conformance: 'Supports', notes: 'Focus is moved to the main content area on every client-side route change.' },
  { sc: '2.4.4', title: 'Link Purpose (In Context)', level: 'A', conformance: 'Supports', notes: 'All links have accessible names via visible text or aria-label.' },
  { sc: '2.5.3', title: 'Label in Name', level: 'A', conformance: 'Supports', notes: 'Accessible names contain or match the visible label text.' },
  { sc: '3.1.1', title: 'Language of Page', level: 'A', conformance: 'Supports', notes: 'html element has lang="en".' },
  { sc: '3.2.1', title: 'On Focus', level: 'A', conformance: 'Supports', notes: 'No context changes occur on focus.' },
  { sc: '3.2.2', title: 'On Input', level: 'A', conformance: 'Supports', notes: 'Form submissions require explicit user action.' },
  { sc: '3.3.1', title: 'Error Identification', level: 'A', conformance: 'Supports', notes: 'Form validation errors are announced via aria-describedby and ARIA live regions.' },
  { sc: '3.3.2', title: 'Labels or Instructions', level: 'A', conformance: 'Supports', notes: 'All form inputs have associated label elements.' },
  { sc: '4.1.1', title: 'Parsing', level: 'A', conformance: 'Supports', notes: 'React renders valid HTML; no duplicate IDs on interactive elements.' },
  { sc: '4.1.2', title: 'Name, Role, Value', level: 'A', conformance: 'Supports', notes: 'Custom widgets expose accessible names, roles, and state via ARIA.' },
  // Level AA
  { sc: '1.2.4', title: 'Captions (Live)', level: 'AA', conformance: 'Not Applicable', notes: 'No live audio/video streaming at this time.' },
  { sc: '1.2.5', title: 'Audio Description (Prerecorded)', level: 'AA', conformance: 'Not Applicable', notes: 'No prerecorded video content delivered by the platform itself.' },
  { sc: '1.3.4', title: 'Orientation', level: 'AA', conformance: 'Supports', notes: 'Content is not locked to a specific display orientation.' },
  { sc: '1.3.5', title: 'Identify Input Purpose', level: 'AA', conformance: 'Supports', notes: 'Login/signup forms use autocomplete attributes (email, current-password, new-password).' },
  { sc: '1.4.3', title: 'Contrast (Minimum)', level: 'AA', conformance: 'Supports', notes: 'All text color tokens meet a minimum 4.5:1 contrast ratio against their backgrounds. Verified via automated CI checks.' },
  { sc: '1.4.4', title: 'Resize Text', level: 'AA', conformance: 'Supports', notes: 'All text can be resized to 200% without loss of content or functionality.' },
  { sc: '1.4.5', title: 'Images of Text', level: 'AA', conformance: 'Supports', notes: 'No images of text are used for decorative or informational purposes.' },
  { sc: '1.4.10', title: 'Reflow', level: 'AA', conformance: 'Supports', notes: 'Content reflows to a single column at 320 CSS pixels. No horizontal scrolling required except for data tables.' },
  { sc: '1.4.11', title: 'Non-text Contrast', level: 'AA', conformance: 'Supports', notes: 'UI component boundaries (buttons, inputs, focus rings) meet 3:1 contrast against adjacent colors.' },
  { sc: '1.4.12', title: 'Text Spacing', level: 'AA', conformance: 'Supports', notes: 'No content or functionality is lost when line-height, letter-spacing, word-spacing, and paragraph spacing overrides are applied.' },
  { sc: '1.4.13', title: 'Content on Hover or Focus', level: 'AA', conformance: 'Supports', notes: 'Tooltips triggered by hover or focus can be dismissed and are persistent until dismissed.' },
  { sc: '2.4.5', title: 'Multiple Ways', level: 'AA', conformance: 'Supports', notes: 'Course content is reachable via sidebar navigation, search, and direct URL.' },
  { sc: '2.4.6', title: 'Headings and Labels', level: 'AA', conformance: 'Supports', notes: 'Headings describe page sections; form labels describe their inputs.' },
  { sc: '2.4.7', title: 'Focus Visible', level: 'AA', conformance: 'Supports', notes: 'All interactive elements have a visible 2px focus ring using the browser default or a custom ring style.' },
  { sc: '3.1.2', title: 'Language of Parts', level: 'AA', conformance: 'Not Applicable', notes: 'Content is English-only at this time (multilingual support planned in plan 11.1).' },
  { sc: '3.2.3', title: 'Consistent Navigation', level: 'AA', conformance: 'Supports', notes: 'Navigation is consistent across all pages.' },
  { sc: '3.2.4', title: 'Consistent Identification', level: 'AA', conformance: 'Supports', notes: 'Components with the same functionality are identified consistently.' },
  { sc: '3.3.3', title: 'Error Suggestion', level: 'AA', conformance: 'Supports', notes: 'Validation errors include actionable descriptions of how to fix the input.' },
  { sc: '3.3.4', title: 'Error Prevention (Legal, Financial, Data)', level: 'AA', conformance: 'Supports', notes: 'Destructive actions (delete course, remove student) require confirmation dialogs.' },
  { sc: '4.1.3', title: 'Status Messages', level: 'AA', conformance: 'Supports', notes: 'Toast notifications and form status messages are announced via ARIA live regions (role="status" / role="alert").' },
]

function conformanceBadgeClass(level: ConformanceLevel): string {
  switch (level) {
    case 'Supports':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
    case 'Partially Supports':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300'
    case 'Does Not Support':
      return 'bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300'
    case 'Not Applicable':
      return 'bg-slate-100 text-slate-600 dark:bg-neutral-800 dark:text-neutral-400'
  }
}

export default function AccessibilityConformancePage() {
  useEffect(() => {
    document.title = 'Accessibility Conformance Statement — Lextures'
  }, [])

  const levelA = CRITERIA.filter((c) => c.level === 'A')
  const levelAA = CRITERIA.filter((c) => c.level === 'AA')

  return (
    <div className="min-h-dvh bg-slate-50 text-slate-900 dark:bg-neutral-950 dark:text-neutral-100">
      <header className="border-b border-slate-200 bg-white px-4 py-4 dark:border-neutral-800 dark:bg-neutral-900 sm:px-6 print:hidden">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3">
          <Link to="/login" className="inline-flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-neutral-200">
            <BrandLogo className="h-7 w-auto" />
            <span className="sr-only">Lextures home</span>
          </Link>
          <nav aria-label="Legal" className="flex flex-wrap gap-3 text-sm">
            <Link to="/privacy" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Privacy</Link>
            <Link to="/terms" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Terms</Link>
            <Link to="/trust" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Trust</Link>
            <Link to="/login" className="text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100">Sign in</Link>
          </nav>
        </div>
      </header>

      <main id="main-content" className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:py-12">
        <div className="mb-10">
          <h1 className="text-3xl font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
            Accessibility Conformance Statement
          </h1>
          <p className="mt-2 text-base text-slate-600 dark:text-neutral-400">
            Lextures strives to conform to the{' '}
            <a
              href="https://www.w3.org/TR/WCAG21/"
              className="text-indigo-700 underline underline-offset-2 dark:text-indigo-300"
              target="_blank"
              rel="noreferrer"
            >
              Web Content Accessibility Guidelines (WCAG) 2.1
            </a>{' '}
            at Level AA, as required by Section 508 of the Rehabilitation Act (36 CFR Part 1194) and
            EN 301 549 for public-sector procurement.
          </p>
          <p className="mt-2 text-sm text-slate-500 dark:text-neutral-500">
            Last updated: May 27, 2026. This statement is reviewed and updated annually.
            Questions? Email{' '}
            <a href="mailto:accessibility@lextures.com" className="text-indigo-700 underline dark:text-indigo-300">
              accessibility@lextures.com
            </a>
            .
          </p>
        </div>

        <div className="mb-8 rounded-lg border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
          <h2 className="mb-3 text-lg font-semibold text-slate-900 dark:text-neutral-50">
            Conformance Summary
          </h2>
          <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {[
              { label: 'Standard', value: 'WCAG 2.1 AA' },
              { label: 'Conformance Level', value: 'AA (target)' },
              { label: 'Evaluation Method', value: 'Axe-core automated + manual review' },
              { label: 'Applicable Laws', value: 'Section 508 / EN 301 549 / ADA Title II' },
            ].map(({ label, value }) => (
              <div key={label}>
                <dt className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                  {label}
                </dt>
                <dd className="mt-1 text-sm font-medium text-slate-900 dark:text-neutral-100">{value}</dd>
              </div>
            ))}
          </dl>
        </div>

        <div className="mb-8 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-300">
          <strong>Status note:</strong> Lextures is an active conformance program (plan 10.7). The
          items marked "Partially Supports" represent known gaps that are actively being remediated.
          Automated axe-core checks run on every pull request to prevent regressions.
        </div>

        <section aria-labelledby="level-a-heading" className="mb-10">
          <h2 id="level-a-heading" className="mb-3 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            WCAG 2.1 Level A Success Criteria
          </h2>
          <CriteriaTable criteria={levelA} />
        </section>

        <section aria-labelledby="level-aa-heading" className="mb-10">
          <h2 id="level-aa-heading" className="mb-3 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            WCAG 2.1 Level AA Success Criteria
          </h2>
          <CriteriaTable criteria={levelAA} />
        </section>

        <section aria-labelledby="contact-heading" className="mb-10">
          <h2 id="contact-heading" className="mb-3 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            Feedback &amp; Assistance
          </h2>
          <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-300">
            <p>
              If you encounter an accessibility barrier on Lextures, please contact us:
            </p>
            <ul className="list-disc pl-6 space-y-1">
              <li>
                Email:{' '}
                <a href="mailto:accessibility@lextures.com" className="text-indigo-700 underline dark:text-indigo-300">
                  accessibility@lextures.com
                </a>
              </li>
              <li>We aim to respond to accessibility inquiries within 2 business days.</li>
              <li>
                For urgent accommodation needs, please also contact your institution's IT or
                accessibility services office.
              </li>
            </ul>
          </div>
        </section>
      </main>

      <footer className="mt-12 border-t border-slate-200 px-4 py-6 text-center text-xs text-slate-500 dark:border-neutral-800 dark:text-neutral-500 print:hidden">
        <p>
          &copy; {new Date().getFullYear()} Lextures, Inc. &middot;{' '}
          <Link to="/privacy" className="underline-offset-2 hover:underline">
            Privacy Policy
          </Link>{' '}
          &middot;{' '}
          <Link to="/terms" className="underline-offset-2 hover:underline">
            Terms of Service
          </Link>{' '}
          &middot;{' '}
          <Link to="/trust" className="underline-offset-2 hover:underline">
            Trust Center
          </Link>
        </p>
      </footer>
    </div>
  )
}

function CriteriaTable({ criteria }: { criteria: Criterion[] }) {
  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200 dark:border-neutral-800">
      <table
        className="min-w-full text-sm border-collapse"
        aria-label="WCAG success criteria conformance"
      >
        <caption className="sr-only">
          WCAG 2.1 success criteria with conformance level and notes
        </caption>
        <thead className="bg-slate-50 dark:bg-neutral-900">
          <tr className="border-b border-slate-200 dark:border-neutral-700">
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400 whitespace-nowrap">
              SC
            </th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400">
              Title
            </th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400 whitespace-nowrap">
              Conformance
            </th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400">
              Notes
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 bg-white dark:divide-neutral-800 dark:bg-neutral-900">
          {criteria.map((c) => (
            <tr key={c.sc} className="hover:bg-slate-50 dark:hover:bg-neutral-800/50">
              <td className="px-4 py-3 font-mono text-xs font-medium text-slate-700 dark:text-neutral-300 whitespace-nowrap">
                {c.sc}
              </td>
              <td className="px-4 py-3 text-slate-800 dark:text-neutral-200">{c.title}</td>
              <td className="px-4 py-3 whitespace-nowrap">
                <span
                  className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${conformanceBadgeClass(c.conformance)}`}
                >
                  {c.conformance}
                </span>
              </td>
              <td className="px-4 py-3 text-slate-600 dark:text-neutral-400">{c.notes}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
