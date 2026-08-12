/**
 * JSON-LD graph merge, validation, and serialisation (SEO.3 FR-1–FR-5).
 */
import type { JsonLdNode } from '../document-head'
import { buildLogoImage, buildFounderPersonStub, buildOrganization } from './organization'
import { buildWebsite } from './website'
import { buildSoftwareApplication } from './software-application'
import { buildBreadcrumbList } from './breadcrumb'

export type SchemaGraph = {
  '@context': 'https://schema.org'
  '@graph': JsonLdNode[]
}

const ABSOLUTE_ID = /^https?:\/\//i

/**
 * Escape + stringify for embedding in `<script type="application/ld+json">` (FR-3).
 */
export function serializeSchemaGraph(nodes: JsonLdNode | JsonLdNode[]): string {
  const list = Array.isArray(nodes) ? nodes : [nodes]
  const graph: SchemaGraph =
    list.length === 1 && list[0]['@graph']
      ? (list[0] as unknown as SchemaGraph)
      : { '@context': 'https://schema.org', '@graph': list }

  // Ensure single context on the envelope only
  if (!graph['@context']) {
    graph['@context'] = 'https://schema.org'
  }

  return JSON.stringify(graph)
    .replace(/</g, '\\u003c')
    .replace(/>/g, '\\u003e')
    .replace(/&/g, '\\u0026')
    .replace(/\u2028/g, '\\u2028')
    .replace(/\u2029/g, '\\u2029')
}

/**
 * Site-wide entity graph: Organization, logo, founder stub, WebSite, SoftwareApplication.
 * Emitted on every page (FR-6–FR-9).
 */
export function buildSiteWideGraph(siteOrigin?: string): JsonLdNode[] {
  return [
    buildLogoImage(siteOrigin),
    buildOrganization(siteOrigin),
    buildFounderPersonStub(siteOrigin),
    buildWebsite(siteOrigin),
    buildSoftwareApplication(siteOrigin),
  ]
}

/**
 * Merge site-wide + page nodes, dedupe by @id (later wins), strip nested @context.
 */
export function mergeGraph(
  siteWide: JsonLdNode[],
  pageNodes: JsonLdNode[],
): JsonLdNode[] {
  const byId = new Map<string, JsonLdNode>()
  const noId: JsonLdNode[] = []

  for (const raw of [...siteWide, ...pageNodes]) {
    if (!raw || typeof raw !== 'object') continue
    const { ['@context']: _c, ...node } = raw as JsonLdNode & { '@context'?: unknown }
    const id = typeof node['@id'] === 'string' ? node['@id'] : null
    if (id) {
      byId.set(id, node as JsonLdNode)
    } else {
      noId.push(node as JsonLdNode)
    }
  }

  return [...byId.values(), ...noId]
}

/**
 * Collect string @id values and @id references from a node tree.
 */
function walkIds(
  value: unknown,
  defined: Set<string>,
  refs: Set<string>,
  isDefining: boolean,
): void {
  if (value == null) return
  if (Array.isArray(value)) {
    for (const v of value) walkIds(v, defined, refs, isDefining)
    return
  }
  if (typeof value !== 'object') return
  const obj = value as Record<string, unknown>
  if (typeof obj['@id'] === 'string') {
    if (isDefining && obj['@type']) {
      defined.add(obj['@id'])
    } else if (!isDefining || Object.keys(obj).length === 1) {
      // Pure reference { "@id": "..." }
      refs.add(obj['@id'])
    } else if (obj['@type']) {
      defined.add(obj['@id'])
    } else {
      refs.add(obj['@id'])
    }
  }
  for (const [k, v] of Object.entries(obj)) {
    if (k === '@id') continue
    // Nested objects that only have @id are references
    walkIds(v, defined, refs, false)
  }
}

export type GraphValidationError = {
  code: 'missing_id' | 'non_absolute_id' | 'dangling_ref' | 'missing_type' | 'missing_property'
  message: string
  nodeId?: string
}

/**
 * Validate graph for build-time gate (FR-5).
 * - Every node needs @type and absolute @id
 * - @id references must resolve within the graph or be documented cross-page ids
 */
export function validateGraph(
  nodes: JsonLdNode[],
  opts?: {
    /** Absolute @ids defined on other pages that may be referenced (authors, org, …). */
    externalIds?: string[]
  },
): GraphValidationError[] {
  const errors: GraphValidationError[] = []
  const defined = new Set<string>(opts?.externalIds ?? [])
  const refs = new Set<string>()

  for (const node of nodes) {
    if (!node || typeof node !== 'object') continue
    const id = node['@id']
    if (typeof id !== 'string' || !id) {
      errors.push({
        code: 'missing_id',
        message: `Node of type ${String(node['@type'] ?? '?')} is missing @id`,
      })
      continue
    }
    if (!ABSOLUTE_ID.test(id)) {
      errors.push({
        code: 'non_absolute_id',
        message: `@id must be absolute URL: ${id}`,
        nodeId: id,
      })
    }
    if (!node['@type']) {
      errors.push({
        code: 'missing_type',
        message: `Node ${id} is missing @type`,
        nodeId: id,
      })
    }
    defined.add(id)
  }

  for (const node of nodes) {
    walkIds(node, defined, refs, true)
  }

  // Re-collect pure refs more carefully
  const pureRefs = new Set<string>()
  const collectRefs = (value: unknown): void => {
    if (value == null) return
    if (Array.isArray(value)) {
      for (const v of value) collectRefs(v)
      return
    }
    if (typeof value !== 'object') return
    const obj = value as Record<string, unknown>
    const keys = Object.keys(obj)
    if (keys.length === 1 && keys[0] === '@id' && typeof obj['@id'] === 'string') {
      pureRefs.add(obj['@id'])
      return
    }
    for (const [k, v] of Object.entries(obj)) {
      if (k === '@id' || k === '@type' || k === '@context') continue
      collectRefs(v)
    }
  }
  for (const node of nodes) collectRefs(node)

  for (const ref of pureRefs) {
    if (!defined.has(ref)) {
      // Cross-page entity ids that are stable site-wide are OK if they match known prefixes
      // already in site-wide graph on every page — if missing, it's dangling.
      errors.push({
        code: 'dangling_ref',
        message: `Dangling @id reference: ${ref}`,
        nodeId: ref,
      })
    }
  }

  return errors
}

/**
 * Compose full page graph: site-wide + breadcrumbs + page-specific nodes.
 */
export function composePageGraph(opts: {
  path: string
  pageNodes?: JsonLdNode[]
  leafName?: string
  siteOrigin?: string
  /** Skip site-wide (tests). */
  skipSiteWide?: boolean
}): JsonLdNode[] {
  const siteWide = opts.skipSiteWide ? [] : buildSiteWideGraph(opts.siteOrigin)
  const crumbs = buildBreadcrumbList(opts.path, {
    leafName: opts.leafName,
    siteOrigin: opts.siteOrigin,
  })
  const page = [...(opts.pageNodes ?? [])]
  if (crumbs) page.unshift(crumbs)
  return mergeGraph(siteWide, page)
}

/**
 * Extract @type strings for .seo-manifest.json (including nested graph).
 */
export function collectSchemaTypes(jsonLd: unknown): string[] {
  if (!jsonLd) return []
  const types: string[] = []
  const visit = (node: unknown): void => {
    if (!node || typeof node !== 'object') return
    if (Array.isArray(node)) {
      for (const n of node) visit(n)
      return
    }
    const obj = node as Record<string, unknown>
    if (obj['@graph']) {
      visit(obj['@graph'])
      return
    }
    if (obj['@type']) {
      const t = obj['@type']
      if (Array.isArray(t)) types.push(...t.map(String))
      else types.push(String(t))
    }
  }
  visit(jsonLd)
  return types
}

/** Byte length of serialised graph (performance budget ≤ 12 KB). */
export function schemaPayloadBytes(nodes: JsonLdNode[]): number {
  return new TextEncoder().encode(serializeSchemaGraph(nodes)).length
}
