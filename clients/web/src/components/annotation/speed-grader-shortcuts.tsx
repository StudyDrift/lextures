/* eslint-disable react-refresh/only-export-components -- component file exports shortcut helpers + hook */
import { Keyboard } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import { isTypingContextTarget } from '../../lib/keyboard-shortcut-eligible'
import type { ModuleAssignmentSubmissionApi } from '../../lib/courses-api'
import {
  adjacentSubmissionIndex,
  adjacentUngradedSubmissionIndex,
} from './submission-navigator-utils'

export function isSpeedGraderScoreInput(target: EventTarget | null): boolean {
  return (
    target instanceof HTMLInputElement &&
    target.type === 'number' &&
    target.dataset.speedGraderScore === 'true'
  )
}

export function altKeyHint(): string {
  if (typeof navigator === 'undefined') return 'Alt'
  const p = navigator.platform ?? ''
  const ua = navigator.userAgent ?? ''
  const apple = /Mac|iPhone|iPad|iPod/.test(p) || /Mac OS/.test(ua)
  return apple ? '⌥' : 'Alt'
}

function Kbd({ children }: { children: string }) {
  return (
    <kbd className="rounded-md border border-slate-200 bg-slate-50 px-1.5 py-0.5 font-mono text-[10px] font-medium text-slate-600 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
      {children}
    </kbd>
  )
}

export type SpeedGraderShortcutRow = {
  action: string
  keys: string
}

export function speedGraderShortcutRows(): SpeedGraderShortcutRow[] {
  const alt = altKeyHint()
  return [
    { action: 'Next ungraded submission', keys: 'D' },
    { action: 'Previous ungraded submission', keys: 'A' },
    { action: 'Next student', keys: 'K' },
    { action: 'Previous student', keys: 'J' },
    { action: 'Save grade', keys: `${alt}+Enter` },
  ]
}

export function SpeedGraderShortcutsPopover({ disabled }: { disabled?: boolean }) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const buttonId = useId()
  const panelId = useId()
  const rows = speedGraderShortcutRows()

  useEffect(() => {
    if (!open) return
    function onPointerDown(e: PointerEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  return (
    <div ref={rootRef} className="relative shrink-0">
      <button
        id={buttonId}
        type="button"
        disabled={disabled}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-controls={panelId}
        aria-label="SpeedGrader keyboard shortcuts"
        title="Keyboard shortcuts"
        onClick={() => setOpen((prev) => !prev)}
        className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-slate-300 bg-white text-slate-600 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-300 dark:hover:bg-neutral-900"
      >
        <Keyboard className="h-4 w-4" aria-hidden="true" />
      </button>

      {open ? (
        <div
          id={panelId}
          role="dialog"
          aria-labelledby={buttonId}
          className="absolute end-0 top-full z-50 mt-1 w-64 rounded-xl border border-slate-200 bg-white p-3 shadow-lg dark:border-neutral-600 dark:bg-neutral-900"
        >
          <p className="mb-2 text-xs font-semibold text-slate-900 dark:text-neutral-100">
            SpeedGrader shortcuts
          </p>
          <ul className="space-y-2">
            {rows.map((row) => (
              <li key={row.keys} className="flex items-center justify-between gap-3 text-xs">
                <span className="text-slate-600 dark:text-neutral-300">{row.action}</span>
                <Kbd>{row.keys}</Kbd>
              </li>
            ))}
          </ul>
          <p className="mt-3 text-[10px] leading-relaxed text-slate-500 dark:text-neutral-400">
            Navigation keys work from the score field; other text fields block them.
          </p>
        </div>
      ) : null}
    </div>
  )
}

type UseSpeedGraderHotkeysOptions = {
  enabled: boolean
  disabled?: boolean
  submissions: ModuleAssignmentSubmissionApi[]
  index: number
  onIndexChange: (index: number) => void
}

export function useSpeedGraderHotkeys({
  enabled,
  disabled = false,
  submissions,
  index,
  onIndexChange,
}: UseSpeedGraderHotkeysOptions) {
  const navRef = useRef({ submissions, index, onIndexChange })
  navRef.current = { submissions, index, onIndexChange }

  useEffect(() => {
    if (!enabled || disabled) return

    function onKeyDown(e: KeyboardEvent) {
      if (e.defaultPrevented || e.metaKey || e.ctrlKey) return

      const key = e.key.toLowerCase()
      if (!['a', 'd', 'j', 'k'].includes(key)) return
      if (isTypingContextTarget(e.target) && !isSpeedGraderScoreInput(e.target)) return

      const { submissions, index: currentIndex, onIndexChange: setIndex } = navRef.current
      let next: number | null = null

      if (key === 'd') {
        next = adjacentUngradedSubmissionIndex(submissions, currentIndex, 1)
      } else if (key === 'a') {
        next = adjacentUngradedSubmissionIndex(submissions, currentIndex, -1)
      } else if (key === 'k') {
        next = adjacentSubmissionIndex(submissions, currentIndex, 1)
      } else if (key === 'j') {
        next = adjacentSubmissionIndex(submissions, currentIndex, -1)
      }

      if (next == null || next === currentIndex) return
      e.preventDefault()
      setIndex(next)
    }

    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [disabled, enabled, submissions])
}