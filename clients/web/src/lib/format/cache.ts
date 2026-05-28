/** LRU cache for Intl formatter instances (max 20 entries per formatter family). */

const MAX_ENTRIES = 20

export class FormatterCache<T> {
  private readonly map = new Map<string, T>()

  get(key: string, factory: () => T): T {
    const hit = this.map.get(key)
    if (hit !== undefined) {
      this.map.delete(key)
      this.map.set(key, hit)
      return hit
    }
    const created = factory()
    this.map.set(key, created)
    if (this.map.size > MAX_ENTRIES) {
      const oldest = this.map.keys().next().value
      if (oldest !== undefined) {
        this.map.delete(oldest)
      }
    }
    return created
  }

  clear(): void {
    this.map.clear()
  }
}

export function stableOptionsKey(options: Intl.DateTimeFormatOptions | Intl.NumberFormatOptions): string {
  return JSON.stringify(options)
}
