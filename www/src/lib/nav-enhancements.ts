/**
 * Tiny progressive-enhancement for header interactions without React
 * (SEO.4 FR-4 content pages + interactive pages as a baseline).
 *
 * Expects data attributes emitted by Header:
 *   [data-nav-menu-open]  — open mobile menu button
 *   [data-nav-menu-close] — close button
 *   [data-nav-menu]       — mobile dialog panel
 *   [data-nav-audiences]  — audiences toggle
 *   [data-nav-audiences-panel]
 */
export function initNavEnhancements(): void {
  if (typeof document === 'undefined') return

  const openBtn = document.querySelector<HTMLElement>('[data-nav-menu-open]')
  const closeBtn = document.querySelector<HTMLElement>('[data-nav-menu-close]')
  const panel = document.querySelector<HTMLElement>('[data-nav-menu]')
  const mobileAudiencesBtn = document.querySelector<HTMLElement>('[data-nav-mobile-audiences]')
  const mobileAudiencesPanel = document.querySelector<HTMLElement>('[data-nav-mobile-audiences-panel]')

  const setMenu = (open: boolean) => {
    if (!panel) return
    panel.hidden = !open
    document.body.style.overflow = open ? 'hidden' : ''
    openBtn?.setAttribute('aria-expanded', open ? 'true' : 'false')
  }

  openBtn?.addEventListener('click', () => setMenu(true))
  closeBtn?.addEventListener('click', () => setMenu(false))
  panel?.querySelectorAll('a').forEach(a => {
    a.addEventListener('click', () => setMenu(false))
  })

  // Desktop dropdowns, including arrow-key roving and Escape focus return.
  document.querySelectorAll<HTMLElement>('[data-nav-dropdown]').forEach(button => {
    const panel = button.parentElement?.querySelector<HTMLElement>('[data-nav-dropdown-panel]')
    if (!panel) return
    const links = [...panel.querySelectorAll<HTMLAnchorElement>('a')]
    const setOpen = (open: boolean) => { panel.hidden = !open; button.setAttribute('aria-expanded', String(open)) }
    button.addEventListener('click', event => { event.stopPropagation(); setOpen(Boolean(panel.hidden)) })
    button.addEventListener('keydown', event => {
      if (event.key === 'ArrowDown') { event.preventDefault(); setOpen(true); links[0]?.focus() }
      if (event.key === 'Escape') { setOpen(false); button.focus() }
    })
    links.forEach((link, index) => link.addEventListener('keydown', event => {
      if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        event.preventDefault(); links[(index + (event.key === 'ArrowDown' ? 1 : links.length - 1)) % links.length]?.focus()
      }
      if (event.key === 'Escape') { setOpen(false); button.focus() }
    }))
    document.addEventListener('click', event => {
      if (!panel.contains(event.target as Node) && event.target !== button) setOpen(false)
    })
  })

  if (mobileAudiencesBtn && mobileAudiencesPanel) {
    mobileAudiencesBtn.addEventListener('click', () => {
      const open = mobileAudiencesPanel.hidden
      mobileAudiencesPanel.hidden = !open
      mobileAudiencesBtn.setAttribute('aria-expanded', open ? 'true' : 'false')
    })
  }

  // Escape closes menu
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape') setMenu(false)
  })
}
