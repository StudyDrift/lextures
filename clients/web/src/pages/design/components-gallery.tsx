/**
 * UX.2 FR-12 — interactive component gallery at /design/components.
 * Staff tool: every core export with variants, RTL preview, and theme switch.
 */
import { useState } from 'react'
import { Link } from 'react-router-dom'
import {
  applyHighContrast,
  applyUiTheme,
  resolveSemanticTheme,
  type SemanticTheme,
  type UiTheme,
} from '../../lib/ui-theme'
import { Button, SegmentedControl } from '../../components/ui'
import { ComponentsGalleryDemos } from './components-gallery-demos'

const THEME_OPTIONS: { id: SemanticTheme; label: string; base: UiTheme; hc: boolean }[] = [
  { id: 'light', label: 'Light', base: 'light', hc: false },
  { id: 'dark', label: 'Dark', base: 'dark', hc: false },
  { id: 'high-contrast-light', label: 'HC Light', base: 'light', hc: true },
  { id: 'high-contrast-dark', label: 'HC Dark', base: 'dark', hc: true },
]

const NAV_IDS = [
  'button',
  'forms',
  'dialog',
  'menu',
  'tabs',
  'display',
  'feedback',
  'layout',
] as const

export default function ComponentsGalleryPage() {
  const [theme, setTheme] = useState<SemanticTheme>(() =>
    typeof document !== 'undefined'
      ? resolveSemanticTheme(
          document.documentElement.getAttribute('data-theme') === 'dark' ? 'dark' : 'light',
          document.documentElement.classList.contains('high-contrast'),
        )
      : 'light',
  )
  const [rtl, setRtl] = useState(false)

  function applyTheme(id: SemanticTheme) {
    const opt = THEME_OPTIONS.find((t) => t.id === id)!
    applyUiTheme(opt.base)
    applyHighContrast(opt.hc)
    setTheme(id)
  }

  return (
    <div
      className="min-h-screen bg-surface-base text-fg-default"
      dir={rtl ? 'rtl' : 'ltr'}
      data-gallery-root
    >
      <div className="mx-auto max-w-5xl px-4 py-8">
        <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
              Design system
            </p>
            <h1 className="text-2xl font-semibold tracking-tight">Component gallery</h1>
            <p className="mt-1 text-sm text-fg-muted">
              UX.2 core library —{' '}
              <Link className="text-accent-fg underline" to="/design/tokens">
                Tokens
              </Link>
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <SegmentedControl
              label="Theme"
              value={theme}
              onChange={(v) => applyTheme(v as SemanticTheme)}
              options={THEME_OPTIONS.map((t) => ({ value: t.id, label: t.label }))}
              size="sm"
            />
            <Button variant="secondary" size="sm" onClick={() => setRtl((v) => !v)}>
              {rtl ? 'LTR' : 'RTL'}
            </Button>
          </div>
        </div>

        <nav
          aria-label="Component index"
          className="mb-8 flex flex-wrap gap-2 text-xs"
        >
          {NAV_IDS.map((id) => (
            <a
              key={id}
              href={`#${id}`}
              className="rounded-full bg-surface-sunken px-2.5 py-1 font-medium text-fg-muted hover:text-fg-default"
            >
              {id}
            </a>
          ))}
        </nav>

        <ComponentsGalleryDemos />
      </div>
    </div>
  )
}
