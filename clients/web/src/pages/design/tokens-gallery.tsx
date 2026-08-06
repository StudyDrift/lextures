/**
 * UX.1 FR-14 — internal token gallery at /design/tokens.
 * Staff-facing specimen: every semantic token, per-theme value, contrast ratio.
 * See also /design/components (UX.2).
 */
import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  applyUiTheme,
  applyHighContrast,
  type UiTheme,
  resolveSemanticTheme,
  type SemanticTheme,
} from '../../lib/ui-theme'
import { contrastRatio, SEMANTIC_PAIRS } from '../../lib/tokens/contrast'

const THEME_OPTIONS: { id: SemanticTheme; label: string; base: UiTheme; hc: boolean }[] = [
  { id: 'light', label: 'Light', base: 'light', hc: false },
  { id: 'dark', label: 'Dark', base: 'dark', hc: false },
  { id: 'high-contrast-light', label: 'HC Light', base: 'light', hc: true },
  { id: 'high-contrast-dark', label: 'HC Dark', base: 'dark', hc: true },
]

const SURFACE_TOKENS = [
  'surface-base',
  'surface-raised',
  'surface-overlay',
  'surface-sunken',
  'surface-inverse',
] as const

const FG_TOKENS = ['fg-default', 'fg-muted', 'fg-subtle', 'fg-on-accent', 'fg-inverse'] as const
const BORDER_TOKENS = ['border-default', 'border-subtle', 'border-strong', 'border-focus'] as const
const STATUS = ['info', 'success', 'warning', 'danger', 'accent'] as const

function readCssVar(name: string): string {
  if (typeof document === 'undefined') return ''
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

function cssColorToHex(css: string): string | null {
  if (!css) return null
  if (/^#[0-9a-f]{6}$/i.test(css)) return css.toLowerCase()
  // Use canvas to resolve any CSS color
  try {
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = 1
    const ctx = canvas.getContext('2d')
    if (!ctx) return null
    ctx.fillStyle = '#000'
    ctx.fillStyle = css
    const resolved = ctx.fillStyle
    if (typeof resolved === 'string' && resolved.startsWith('#')) {
      if (resolved.length === 4) {
        return `#${resolved[1]}${resolved[1]}${resolved[2]}${resolved[2]}${resolved[3]}${resolved[3]}`
      }
      return resolved.slice(0, 7)
    }
    // rgb(r, g, b)
    const m = /^rgba?\((\d+),\s*(\d+),\s*(\d+)/i.exec(resolved)
    if (m) {
      const hex = (n: string) => Number(n).toString(16).padStart(2, '0')
      return `#${hex(m[1])}${hex(m[2])}${hex(m[3])}`
    }
  } catch {
    /* ignore */
  }
  return null
}

function Swatch({ token, cssVar }: { token: string; cssVar: string }) {
  const value = readCssVar(cssVar)
  return (
    <div className="flex items-center gap-3 rounded-md border border-border-default bg-surface-raised p-3">
      <div
        className="h-10 w-10 shrink-0 rounded-md border border-border-subtle"
        style={{ background: `var(${cssVar})` }}
        aria-hidden
      />
      <div className="min-w-0">
        <p className="font-mono text-sm text-fg-default">{token}</p>
        <p className="truncate font-mono text-xs text-fg-muted" title={value}>
          {value || '—'}
        </p>
        <p className="font-mono text-[10px] text-fg-subtle">{cssVar}</p>
      </div>
    </div>
  )
}

export default function TokensGalleryPage() {
  const [themeId, setThemeId] = useState<SemanticTheme>('light')

  function selectTheme(opt: (typeof THEME_OPTIONS)[number]) {
    applyHighContrast(opt.hc)
    applyUiTheme(opt.base)
    setThemeId(resolveSemanticTheme(opt.base, opt.hc))
  }

  const pairs = useMemo(() => {
    // Force recompute when theme changes
    void themeId
    return SEMANTIC_PAIRS.map((p) => {
      const fgVar = `--lx-${p.fg.replace(/([A-Z])/g, (m) => `-${m.toLowerCase()}`).replace('on-accent', 'on-accent')}`
      // Map JS names to CSS: fg-default → --lx-fg-default
      const toVar = (name: string) => {
        const map: Record<string, string> = {
          'fg-default': '--lx-fg-default',
          'fg-muted': '--lx-fg-muted',
          'fg-subtle': '--lx-fg-subtle',
          'fg-on-accent': '--lx-fg-on-accent',
          'surface-base': '--lx-surface-base',
          'surface-raised': '--lx-surface-raised',
          'surface-sunken': '--lx-surface-sunken',
          'accent-solid': '--lx-accent-solid',
          'info-fg': '--lx-info-fg',
          'info-surface': '--lx-info-surface',
          'success-fg': '--lx-success-fg',
          'success-surface': '--lx-success-surface',
          'warning-fg': '--lx-warning-fg',
          'warning-surface': '--lx-warning-surface',
          'danger-fg': '--lx-danger-fg',
          'danger-surface': '--lx-danger-surface',
          'accent-fg': '--lx-accent-fg',
          'accent-surface': '--lx-accent-surface',
          'border-default': '--lx-border-default',
          'focus-ring': '--lx-focus-ring',
        }
        return map[name] ?? `--lx-${name}`
      }
      const fgHex = cssColorToHex(readCssVar(toVar(p.fg)))
      const bgHex = cssColorToHex(readCssVar(toVar(p.bg)))
      const ratio = fgHex && bgHex ? contrastRatio(fgHex, bgHex) : null
      const pass = ratio != null && ratio >= p.minRatio
      return { ...p, ratio, pass, fgVar }
    })
  }, [themeId])

  return (
    <main className="min-h-screen bg-surface-base text-fg-default">
      <header className="border-b border-border-default bg-surface-raised px-5 py-5 sm:px-8">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wider text-accent-fg">UX.1</p>
            <h1 className="text-2xl font-semibold tracking-tight">Design tokens</h1>
            <p className="mt-1 text-sm text-fg-muted">
              Semantic token gallery — copy names into feature code. See{' '}
              <code className="text-fg-default">docs/design-tokens.md</code>
              {' · '}
              <Link className="text-accent-fg underline" to="/design/components">
                Components
              </Link>
              .
            </p>
          </div>
          <nav className="flex flex-wrap items-center gap-2" aria-label="Theme">
            {THEME_OPTIONS.map((opt) => (
              <button
                key={opt.id}
                type="button"
                onClick={() => selectTheme(opt)}
                className={
                  themeId === opt.id
                    ? 'rounded-md bg-accent-solid px-3 py-1.5 text-sm font-medium text-fg-on-accent'
                    : 'rounded-md border border-border-default bg-surface-raised px-3 py-1.5 text-sm text-fg-default'
                }
              >
                {opt.label}
              </button>
            ))}
            <Link to="/type" className="ms-2 text-sm text-accent-fg underline-offset-2 hover:underline">
              Type specimen
            </Link>
          </nav>
        </div>
      </header>

      <div className="mx-auto max-w-6xl space-y-12 px-5 py-10 sm:px-8">
        <section aria-labelledby="surfaces-heading">
          <h2 id="surfaces-heading" className="mb-4 text-lg font-semibold">
            Surfaces
          </h2>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {SURFACE_TOKENS.map((t) => (
              <Swatch key={t} token={`bg-${t}`} cssVar={`--lx-${t}`} />
            ))}
          </div>
        </section>

        <section aria-labelledby="fg-heading">
          <h2 id="fg-heading" className="mb-4 text-lg font-semibold">
            Foregrounds
          </h2>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {FG_TOKENS.map((t) => (
              <Swatch key={t} token={`text-${t}`} cssVar={`--lx-${t}`} />
            ))}
          </div>
        </section>

        <section aria-labelledby="border-heading">
          <h2 id="border-heading" className="mb-4 text-lg font-semibold">
            Borders
          </h2>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {BORDER_TOKENS.map((t) => (
              <Swatch key={t} token={`border-${t}`} cssVar={`--lx-${t}`} />
            ))}
          </div>
        </section>

        <section aria-labelledby="status-heading">
          <h2 id="status-heading" className="mb-4 text-lg font-semibold">
            Status
          </h2>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {STATUS.flatMap((s) => [
              <Swatch key={`${s}-surface`} token={`${s}-surface`} cssVar={`--lx-${s}-surface`} />,
              <Swatch key={`${s}-fg`} token={`${s}-fg`} cssVar={`--lx-${s}-fg`} />,
              <Swatch key={`${s}-border`} token={`${s}-border`} cssVar={`--lx-${s}-border`} />,
            ])}
          </div>
        </section>

        <section aria-labelledby="contrast-heading">
          <h2 id="contrast-heading" className="mb-4 text-lg font-semibold">
            Contrast pairs ({themeId})
          </h2>
          <div className="overflow-x-auto rounded-lg border border-border-default bg-surface-raised">
            <table className="w-full min-w-[32rem] text-start text-sm">
              <thead className="border-b border-border-subtle bg-surface-sunken text-fg-muted">
                <tr>
                  <th className="px-3 py-2 font-medium">Foreground</th>
                  <th className="px-3 py-2 font-medium">Background</th>
                  <th className="px-3 py-2 font-medium">Ratio</th>
                  <th className="px-3 py-2 font-medium">AA</th>
                  <th className="px-3 py-2 font-medium">Usage</th>
                </tr>
              </thead>
              <tbody>
                {pairs.map((p) => (
                  <tr key={`${p.fg}-${p.bg}`} className="border-b border-border-subtle last:border-0">
                    <td className="px-3 py-2 font-mono text-xs">{p.fg}</td>
                    <td className="px-3 py-2 font-mono text-xs">{p.bg}</td>
                    <td className="px-3 py-2 font-mono tabular-nums">
                      {p.ratio != null ? `${p.ratio.toFixed(2)}:1` : '—'}
                    </td>
                    <td className="px-3 py-2">
                      <span
                        className={
                          p.pass
                            ? 'inline-flex items-center gap-1 text-success-fg'
                            : 'inline-flex items-center gap-1 text-danger-fg'
                        }
                      >
                        <span aria-hidden>{p.pass ? '✓' : '✗'}</span>
                        <span className="sr-only">{p.pass ? 'Pass' : 'Fail'}</span>
                        {p.pass ? 'Pass' : 'Fail'}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-fg-muted">{p.usage}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <section aria-labelledby="usage-heading" className="pb-16">
          <h2 id="usage-heading" className="mb-4 text-lg font-semibold">
            Authoring
          </h2>
          <pre className="overflow-x-auto rounded-lg border border-border-default bg-surface-sunken p-4 text-xs text-fg-default">
{`// Semantic only — never slate-*/neutral-* in feature code
<div className="bg-surface-raised text-fg-default border border-border-default">
  <p className="text-fg-muted">Secondary copy</p>
  <button className="bg-accent-solid text-fg-on-accent">Save</button>
</div>`}
          </pre>
        </section>
      </div>
    </main>
  )
}
