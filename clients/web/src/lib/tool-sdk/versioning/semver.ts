export type SemVer = { major: number; minor: number; patch: number }

export function parseSemVer(v: string): SemVer {
  const cleaned = v.trim().split(/[-+]/)[0] ?? ''
  const parts = cleaned.split('.')
  if (parts.length !== 3) throw new Error(`invalid semver: ${v}`)
  const [major, minor, patch] = parts.map((p) => Number(p))
  if (![major, minor, patch].every((n) => Number.isInteger(n) && n >= 0)) {
    throw new Error(`invalid semver: ${v}`)
  }
  return { major, minor, patch }
}

export function compareSemVer(a: SemVer, b: SemVer): number {
  if (a.major !== b.major) return a.major < b.major ? -1 : 1
  if (a.minor !== b.minor) return a.minor < b.minor ? -1 : 1
  if (a.patch !== b.patch) return a.patch < b.patch ? -1 : 1
  return 0
}

export function resolveWithinMajor(pinned: string, published: string[]): string {
  const pin = parseSemVer(pinned)
  const cands = published
    .map((raw) => {
      try {
        return { raw, v: parseSemVer(raw) }
      } catch {
        return null
      }
    })
    .filter((c): c is { raw: string; v: SemVer } => !!c && c.v.major === pin.major)
    .sort((a, b) => compareSemVer(b.v, a.v))
  if (cands.length === 0) throw new Error(`no published version within major of ${pinned}`)
  const preferred = cands.find((c) => compareSemVer(c.v, pin) >= 0)
  return (preferred ?? cands[0]).raw
}
