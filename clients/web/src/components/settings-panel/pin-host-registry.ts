/**
 * Imperative registry of portal host elements for pinned setting rows.
 * Kept outside React state so dnd-kit re-renders / ref detach-attach cycles
 * do not infinite-loop setState (PS.3 move-not-duplicate portals).
 */

type Listener = () => void

const hosts = new Map<string, HTMLElement>()
const listeners = new Set<Listener>()

function emit(): void {
  for (const l of listeners) {
    try {
      l()
    } catch {
      // ignore
    }
  }
}

export function setPinHost(settingId: string, el: HTMLElement | null): void {
  if (el) {
    if (hosts.get(settingId) === el) return
    hosts.set(settingId, el)
  } else {
    if (!hosts.has(settingId)) return
    hosts.delete(settingId)
  }
  emit()
}

export function getPinHost(settingId: string): HTMLElement | null {
  return hosts.get(settingId) ?? null
}

export function subscribePinHosts(listener: Listener): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

/** Test-only. */
export function __clearPinHostsForTests(): void {
  hosts.clear()
  emit()
}
