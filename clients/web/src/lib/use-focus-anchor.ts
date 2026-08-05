/**
 * Focus-anchor runtime (CC.8).
 *
 * - `useFocusAnchorRuntime()` — mount once under the course layout; runs when
 *   `?focus=` is present, then strips the param.
 * - `useAnchorRef(id)` / `<Anchor id>` — attach `data-focus-anchor`.
 * - `hrefForTarget` — build a shareable link with substituted params + focus.
 * - `registerEntityRevealer` — virtualised lists register a reveal callback.
 */

import {
  useEffect,
  useRef,
  type RefObject,
} from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { announce } from './a11y/announcer'
import {
  parseCompositeAnchor,
  resolveFocusAnchor,
  type FocusAnchor,
  type FocusContainer,
} from './focus-anchors'
import { prefersReducedMotion } from './motion'

export const FOCUS_QUERY_PARAM = 'focus'
export const FOCUS_ENTITY_QUERY_PARAM = 'focusEntity'

const OBSERVER_TIMEOUT_MS = 5_000
const HIGHLIGHT_MS = 4_000
const HIGHLIGHT_FADE_MS = 200
const SCROLL_CLEAR_PX = 200

const HIGHLIGHT_CLASS = 'lx-focus-anchor-highlight'
const CHIP_CLASS = 'lx-focus-anchor-chip'
const CHIP_ATTR = 'data-focus-anchor-chip'

/** CSS.escape polyfill for older engines / test envs. */
export function cssEscape(value: string): string {
  if (typeof CSS !== 'undefined' && typeof CSS.escape === 'function') {
    return CSS.escape(value)
  }
  return value.replace(/[^a-zA-Z0-9_-]/g, (ch) => `\\${ch}`)
}

// ── Entity revealers (virtualised lists) ─────────────────────────────────────

type RevealFn = (entityId: string) => void | Promise<void>

const entityRevealers = new Map<string, RevealFn>()

/**
 * Register a revealer for entity anchors on a route key (e.g. `modules`).
 * Returns an unregister function — call from effect cleanup.
 */
export function registerEntityRevealer(
  routeKey: string,
  reveal: RevealFn,
): () => void {
  entityRevealers.set(routeKey, reveal)
  return () => {
    if (entityRevealers.get(routeKey) === reveal) {
      entityRevealers.delete(routeKey)
    }
  }
}

function routeKeyFromPath(pathname: string): string {
  // /courses/X/modules → modules; /courses/X/settings/outcomes → outcomes
  const parts = pathname.split('/').filter(Boolean)
  if (parts[0] !== 'courses' || parts.length < 3) return parts[parts.length - 1] ?? ''
  if (parts[2] === 'settings' && parts[3]) return parts[3]
  return parts[2] ?? ''
}

// ── href builder ─────────────────────────────────────────────────────────────

export type NavTargetLike = {
  route?: string | null
  anchor?: string | null
  entityKey?: string | null
}

/**
 * Build a shareable path for a checklist / help-centre target.
 * Substitutes `{courseCode}`, `{itemId}`, and any extra `params`.
 * Appends `?focus=` / `&focusEntity=` when an anchor is present and resolvable.
 */
export function hrefForTarget(
  t: NavTargetLike | null | undefined,
  params?: Record<string, string>,
): string {
  if (!t?.route) return ''
  let path = t.route
  const all: Record<string, string> = { ...(params ?? {}) }
  if (t.entityKey && !all.itemId) {
    all.itemId = t.entityKey
  }
  for (const [key, value] of Object.entries(all)) {
    path = path.split(`{${key}}`).join(encodeURIComponent(value))
  }
  // Leave unsubstituted tokens as-is only if missing — still usable for templates.

  const rawAnchor = t.anchor?.trim()
  if (!rawAnchor) return path

  const { baseId, entityId: fromComposite } = parseCompositeAnchor(rawAnchor)
  const entityId = t.entityKey?.trim() || fromComposite
  const resolved = resolveFocusAnchor(baseId)
  if (!resolved) {
    // Unknown anchor: plain navigation (FR-8)
    return path
  }

  const qs = new URLSearchParams()
  qs.set(FOCUS_QUERY_PARAM, resolved.id)
  if (entityId && (resolved.kind === 'entity' || fromComposite)) {
    qs.set(FOCUS_ENTITY_QUERY_PARAM, entityId)
  } else if (entityId && resolved.kind !== 'entity') {
    // Entity key may still be useful for editor routes (already in path);
    // only attach focusEntity for entity-kind anchors.
  }
  // For entity-kind always attach when we have an id
  if (resolved.kind === 'entity' && entityId) {
    qs.set(FOCUS_ENTITY_QUERY_PARAM, entityId)
  }

  const sep = path.includes('?') ? '&' : '?'
  return `${path}${sep}${qs.toString()}`
}

// ── DOM helpers ──────────────────────────────────────────────────────────────

function findAnchorElement(
  anchorId: string,
  entityId?: string | null,
): HTMLElement | null {
  const escaped = cssEscape(anchorId)
  if (entityId) {
    const entityEsc = cssEscape(entityId)
    const withEntity = document.querySelector(
      `[data-focus-anchor="${escaped}"][data-focus-entity="${entityEsc}"]`,
    ) as HTMLElement | null
    if (withEntity) return withEntity
  }
  // PS.1 setting rows share the same address space
  const setting = document.querySelector(
    `[data-setting-row="${escaped}"]`,
  ) as HTMLElement | null
  if (setting) return setting

  return document.querySelector(
    `[data-focus-anchor="${escaped}"]`,
  ) as HTMLElement | null
}

function openContainer(container: FocusContainer | undefined): void {
  if (!container) return
  const attr =
    container.type === 'accordion'
      ? 'data-focus-accordion'
      : container.type === 'tab'
        ? 'data-focus-tab'
        : 'data-focus-section'
  const el = document.querySelector(
    `[${attr}="${cssEscape(container.id)}"]`,
  ) as HTMLElement | null
  if (!el) return

  if (el instanceof HTMLDetailsElement) {
    el.open = true
    return
  }
  // Custom accordion: click a closed summary/button if aria-expanded=false
  const toggle =
    el.matches('button,[role="button"],summary')
      ? el
      : (el.querySelector('summary, button[aria-expanded="false"]') as HTMLElement | null)
  if (toggle && toggle.getAttribute('aria-expanded') === 'false') {
    toggle.click()
  }
  // Tab: select via click or aria
  if (container.type === 'tab') {
    const tab =
      el.getAttribute('role') === 'tab'
        ? el
        : (el.querySelector('[role="tab"]') as HTMLElement | null)
    if (tab && tab.getAttribute('aria-selected') !== 'true') {
      tab.click()
    }
  }
}

function focusableWithin(root: HTMLElement): HTMLElement {
  if (
    root.matches(
      'a[href],button,input,select,textarea,[tabindex]:not([tabindex="-1"])',
    )
  ) {
    return root
  }
  const inner = root.querySelector(
    'a[href],button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])',
  ) as HTMLElement | null
  if (inner) return inner
  // Region anchors: make wrapper programmatically focusable
  if (!root.hasAttribute('tabindex')) {
    root.setAttribute('tabindex', '-1')
  }
  return root
}

function applyHighlight(el: HTMLElement, label: string): () => void {
  el.classList.add(HIGHLIGHT_CLASS)
  let chip = el.querySelector(`:scope > [${CHIP_ATTR}]`) as HTMLElement | null
  if (!chip) {
    chip = document.createElement('span')
    chip.setAttribute(CHIP_ATTR, 'true')
    chip.className = CHIP_CLASS
    chip.textContent = 'Here'
    chip.setAttribute('aria-hidden', 'true')
    // Prefer first child position
    el.style.position = el.style.position || 'relative'
    el.appendChild(chip)
  }

  const reduced = prefersReducedMotion()
  let cleared = false
  let timer: ReturnType<typeof setTimeout> | undefined
  let fadeTimer: ReturnType<typeof setTimeout> | undefined
  let startScrollY = window.scrollY

  const clear = () => {
    if (cleared) return
    cleared = true
    if (timer) clearTimeout(timer)
    if (fadeTimer) clearTimeout(fadeTimer)
    window.removeEventListener('keydown', onInteract, true)
    window.removeEventListener('pointerdown', onInteract, true)
    window.removeEventListener('scroll', onScroll, true)
    if (reduced) {
      el.classList.remove(HIGHLIGHT_CLASS)
      chip?.remove()
    } else {
      el.classList.add(`${HIGHLIGHT_CLASS}--fading`)
      fadeTimer = setTimeout(() => {
        el.classList.remove(HIGHLIGHT_CLASS, `${HIGHLIGHT_CLASS}--fading`)
        chip?.remove()
      }, HIGHLIGHT_FADE_MS)
    }
  }

  const onInteract = () => clear()
  const onScroll = () => {
    if (Math.abs(window.scrollY - startScrollY) > SCROLL_CLEAR_PX) clear()
  }

  window.addEventListener('keydown', onInteract, true)
  window.addEventListener('pointerdown', onInteract, true)
  window.addEventListener('scroll', onScroll, true)

  timer = setTimeout(clear, HIGHLIGHT_MS)

  void label
  return clear
}

function announcementFor(anchor: FocusAnchor): string {
  return `${anchor.label} — this is the setting from your checklist.`
}

async function runFocusSequence(
  anchor: FocusAnchor,
  entityId: string | null,
  signal: { cancelled: boolean },
): Promise<boolean> {
  // 1. Entity reveal for virtualised lists
  if (anchor.kind === 'entity' && entityId) {
    const key = routeKeyFromPath(window.location.pathname)
    const reveal = entityRevealers.get(key) ?? entityRevealers.get(anchor.id.split('.')[0] ?? '')
    if (reveal) {
      try {
        await reveal(entityId)
      } catch {
        /* land on list (FR reliability) */
      }
    }
  }

  if (signal.cancelled) return false

  // 2. Open declared container
  openContainer(anchor.container)

  // 3. Bounded MutationObserver for the element
  const found = await waitForElement(anchor.id, entityId, OBSERVER_TIMEOUT_MS, signal)
  if (!found || signal.cancelled) return false

  // 4. Scroll + focus + highlight
  const reduced = prefersReducedMotion()
  found.scrollIntoView({
    block: 'center',
    behavior: reduced ? 'auto' : 'smooth',
  })

  const target = focusableWithin(found)
  try {
    target.focus({ preventScroll: true })
  } catch {
    /* ignore */
  }

  applyHighlight(found, anchor.label)
  announce(announcementFor(anchor), 'polite')

  // Telemetry hook (CC.10 dictionary — fire-and-forget console in dev)
  if (import.meta.env.DEV) {
    // eslint-disable-next-line no-console
    console.debug('[focus-anchor] checklist_target_navigated', {
      anchorId: anchor.id,
      resolved: true,
    })
  } else {
    try {
      window.dispatchEvent(
        new CustomEvent('checklist_target_navigated', {
          detail: { anchorId: anchor.id, resolved: true },
        }),
      )
    } catch {
      /* ignore */
    }
  }

  return true
}

function waitForElement(
  anchorId: string,
  entityId: string | null,
  timeoutMs: number,
  signal: { cancelled: boolean },
): Promise<HTMLElement | null> {
  const existing = findAnchorElement(anchorId, entityId)
  if (existing) return Promise.resolve(existing)

  return new Promise((resolve) => {
    const root = document.getElementById('main-content') ?? document.body
    let settled = false
    const done = (el: HTMLElement | null) => {
      if (settled) return
      settled = true
      observer.disconnect()
      clearTimeout(timer)
      resolve(el)
    }

    const observer = new MutationObserver(() => {
      if (signal.cancelled) {
        done(null)
        return
      }
      const el = findAnchorElement(anchorId, entityId)
      if (el) done(el)
    })
    observer.observe(root, { childList: true, subtree: true })

    const timer = setTimeout(() => done(null), timeoutMs)
  })
}

function stripFocusParams(search: string): string {
  const params = new URLSearchParams(search)
  if (!params.has(FOCUS_QUERY_PARAM) && !params.has(FOCUS_ENTITY_QUERY_PARAM)) {
    return search
  }
  params.delete(FOCUS_QUERY_PARAM)
  params.delete(FOCUS_ENTITY_QUERY_PARAM)
  const next = params.toString()
  return next ? `?${next}` : ''
}

/**
 * Mount once in the course layout. No-op when `?focus=` is absent (common path).
 */
export function useFocusAnchorRuntime(): void {
  const location = useLocation()
  const navigate = useNavigate()
  /** Dedupes re-entry after we strip `?focus=` (same navigation must not re-fire). */
  const handledKeyRef = useRef<string | null>(null)
  /** Cancels an in-flight sequence when a newer focus navigation starts or we unmount. */
  const activeSignalRef = useRef<{ cancelled: boolean } | null>(null)

  useEffect(() => {
    const params = new URLSearchParams(location.search)
    const rawFocus = params.get(FOCUS_QUERY_PARAM)
    if (!rawFocus) return // zero work on the common path (NFR)

    const focusEntity = params.get(FOCUS_ENTITY_QUERY_PARAM)
    const handleKey = `${location.pathname}?${rawFocus}&${focusEntity ?? ''}`
    if (handledKeyRef.current === handleKey) return
    handledKeyRef.current = handleKey

    // Cancel any previous sequence (new deep-link supersedes).
    if (activeSignalRef.current) activeSignalRef.current.cancelled = true

    const { baseId, entityId: fromComposite } = parseCompositeAnchor(rawFocus)
    const entityId = focusEntity || fromComposite || null
    const resolved = resolveFocusAnchor(baseId)

    // Strip params via history.replace so refresh/back does not re-fire (FR-7).
    const cleanedSearch = stripFocusParams(location.search)
    navigate(
      { pathname: location.pathname, search: cleanedSearch, hash: location.hash },
      { replace: true },
    )

    if (!resolved) {
      if (import.meta.env.DEV) {
        // eslint-disable-next-line no-console
        console.warn(`[focus-anchor] Unknown focus anchor "${rawFocus}" — navigation only.`)
      }
      try {
        window.dispatchEvent(
          new CustomEvent('checklist_target_navigated', {
            detail: { anchorId: rawFocus, resolved: false },
          }),
        )
      } catch {
        /* ignore */
      }
      return
    }

    const signal = { cancelled: false }
    activeSignalRef.current = signal
    void runFocusSequence(resolved, entityId, signal).then((ok) => {
      if (!ok && !signal.cancelled && import.meta.env.DEV) {
        // eslint-disable-next-line no-console
        console.warn(
          `[focus-anchor] Element for "${resolved.id}" not found within ${OBSERVER_TIMEOUT_MS}ms.`,
        )
      }
    })
  }, [location.pathname, location.search, location.hash, navigate])

  // Cancel on unmount only
  useEffect(() => {
    return () => {
      if (activeSignalRef.current) activeSignalRef.current.cancelled = true
    }
  }, [])
}

/**
 * Attach `data-focus-anchor` via a ref. Prefer for native elements.
 * For entity rows pass `entityId` so the runtime can match `data-focus-entity`.
 */
export function useAnchorRef<T extends HTMLElement>(
  anchorId: string,
  entityId?: string,
): RefObject<T | null> {
  const ref = useRef<T | null>(null)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    el.setAttribute('data-focus-anchor', anchorId)
    if (entityId) el.setAttribute('data-focus-entity', entityId)
    else el.removeAttribute('data-focus-entity')
  }, [anchorId, entityId])
  return ref
}
