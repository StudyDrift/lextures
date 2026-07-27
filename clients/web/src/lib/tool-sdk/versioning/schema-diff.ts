export type BumpKind = 'none' | 'patch' | 'minor' | 'major'

export type SchemaDiffFinding = { path: string; kind: BumpKind; note: string }

const RANK: Record<BumpKind, number> = { none: 0, patch: 1, minor: 2, major: 3 }

type JsonSchema = {
  type?: string
  properties?: Record<string, JsonSchema>
  required?: string[]
  enum?: unknown[]
  title?: string
  description?: string
  [key: string]: unknown
}

export function classifySchemaDiff(
  oldSchema: unknown,
  newSchema: unknown,
): { kind: BumpKind; findings: SchemaDiffFinding[] } {
  const findings = diffNode('$', oldSchema as JsonSchema, newSchema as JsonSchema)
  let kind: BumpKind = 'none'
  for (const f of findings) {
    if (RANK[f.kind] > RANK[kind]) kind = f.kind
  }
  return { kind, findings }
}

export function assertVersionCoversSchemaDiff(
  fromVer: string,
  toVer: string,
  oldSchema: unknown,
  newSchema: unknown,
): void {
  const declared = classifyVersionBump(fromVer, toVer)
  const { kind: required, findings } = classifySchemaDiff(oldSchema, newSchema)
  if (RANK[declared] >= RANK[required]) return
  const paths = findings
    .filter((f) => RANK[f.kind] >= RANK[required])
    .map((f) => `${f.path} (${f.kind}: ${f.note})`)
  throw new Error(
    `schema diff requires ${required} bump but version ${fromVer} → ${toVer} is only ${declared}; offending: ${paths.join('; ')}`,
  )
}

function classifyVersionBump(fromVer: string, toVer: string): BumpKind {
  const a = parseParts(fromVer)
  const b = parseParts(toVer)
  if (a.major === b.major && a.minor === b.minor && a.patch === b.patch) return 'none'
  if (b.major > a.major) return 'major'
  if (b.major < a.major) throw new Error('version decreased')
  if (b.minor > a.minor) return 'minor'
  if (b.minor < a.minor) throw new Error('version decreased')
  if (b.patch > a.patch) return 'patch'
  throw new Error('version decreased')
}

function parseParts(v: string) {
  const [maj, min, pat] = v.trim().split(/[-+]/)[0].split('.').map(Number)
  return { major: maj, minor: min, patch: pat }
}

function diffNode(path: string, oldV: JsonSchema | undefined, newV: JsonSchema | undefined): SchemaDiffFinding[] {
  const out: SchemaDiffFinding[] = []
  if (!oldV || !newV || typeof oldV !== 'object' || typeof newV !== 'object') {
    if (JSON.stringify(oldV) !== JSON.stringify(newV)) {
      out.push({ path, kind: 'major', note: 'type or value changed' })
    }
    return out
  }
  if (oldV.type && newV.type && oldV.type !== newV.type) {
    out.push({ path: `${path}.type`, kind: 'major', note: `${oldV.type} → ${newV.type}` })
  }
  const oldProps = oldV.properties ?? {}
  const newProps = newV.properties ?? {}
  const oldReq = new Set(oldV.required ?? [])
  const newReq = new Set(newV.required ?? [])
  for (const name of Object.keys(oldProps)) {
    const p = `${path}.properties.${name}`
    if (!(name in newProps)) {
      out.push({ path: p, kind: 'major', note: 'field removed' })
      continue
    }
    out.push(...diffNode(p, oldProps[name], newProps[name]))
    if (!oldReq.has(name) && newReq.has(name)) {
      out.push({ path: p, kind: 'major', note: 'became required' })
    }
  }
  for (const name of Object.keys(newProps)) {
    if (name in oldProps) continue
    const p = `${path}.properties.${name}`
    out.push({
      path: p,
      kind: newReq.has(name) ? 'major' : 'minor',
      note: newReq.has(name) ? 'new required field' : 'additive optional field',
    })
  }
  return out
}
