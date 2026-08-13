/**
 * Deterministic mermaid flowchart → HTML for marketing content.
 * Supports graph/flowchart TD|LR with subgraphs, nodes, edges, and style fills.
 * Unknown mermaid dialects stay as a language-mermaid code block.
 */

export type MermaidNode = { id: string; label: string; tone: string }
export type MermaidSubgraph = { title: string; ids: string[] }

const FENCE_RE = /^```[ \t]*mermaid[ \t]*\r?\n([\s\S]*?)^```[ \t]*$/gim
const HEADER_RE = /^(graph|flowchart)\s+(TD|TB|BT|LR|RL)?\s*$/i
const STYLE_RE = /^style\s+(\S+)\s+(.+)$/i
const FILL_RE = /fill\s*:\s*#([0-9a-fA-F]{3,8})/i

export function extractMermaidFences(source: string): { markdown: string; replacements: Map<string, string> } {
  const replacements = new Map<string, string>()
  let sequence = 0
  const markdown = source.replace(FENCE_RE, (_whole, body: string) => {
    const token = `CONTENTMERMAID${sequence++}TOKEN`
    const html = renderMermaidBlock(body)
    replacements.set(`<p>${token}</p>`, html)
    replacements.set(token, html)
    return token
  })
  return { markdown, replacements }
}

export function renderMermaidBlock(source: string): string {
  const graph = parseMermaidGraph(source)
  if (!graph) return mermaidCodeBlock(source)
  return renderMermaidHTML(graph)
}

export function parseMermaidGraph(source: string): {
  direction: 'td' | 'lr'
  nodes: MermaidNode[]
  subgraphs: MermaidSubgraph[]
} | null {
  const lines = String(source || '').replace(/\r\n/g, '\n').split('\n')
  let start = 0
  while (start < lines.length && !lines[start].trim()) start++
  if (start >= lines.length) return null
  const header = lines[start].trim().match(HEADER_RE)
  if (!header) return null

  const direction: 'td' | 'lr' = /^(LR|RL)$/i.test(header[2] || 'TD') ? 'lr' : 'td'
  const nodeMap = new Map<string, MermaidNode>()
  const subgraphs: MermaidSubgraph[] = []
  let current: MermaidSubgraph | null = null

  const ensure = (id: string, label?: string) => {
    const existing = nodeMap.get(id)
    if (!existing) {
      const node = { id, label: label || id, tone: 'neutral' }
      nodeMap.set(id, node)
    } else if (label && label !== id) {
      existing.label = label
    }
    if (current && !current.ids.includes(id)) current.ids.push(id)
    return nodeMap.get(id)!
  }

  for (let i = start + 1; i < lines.length; i++) {
    const raw = lines[i]
    const line = raw.trim()
    if (!line || line.startsWith('%%')) continue
    if (/^end$/i.test(line)) {
      current = null
      continue
    }
    const sub = line.match(/^subgraph\s+(.+)$/i)
    if (sub) {
      current = { title: subgraphTitle(sub[1]), ids: [] }
      subgraphs.push(current)
      continue
    }
    const style = line.match(STYLE_RE)
    if (style) {
      const node = ensure(style[1])
      node.tone = toneFromFill(style[2])
      continue
    }
    if (/^(classDef|class|linkStyle|click)\b/i.test(line)) continue
    const flow = parseFlowLine(line)
    if (!flow) continue
    for (const node of flow.nodes) ensure(node.id, node.label)
  }

  if (nodeMap.size === 0) return null
  if (subgraphs.length === 0) {
    subgraphs.push({ title: '', ids: [...nodeMap.keys()] })
  } else {
    const claimed = new Set(subgraphs.flatMap(group => group.ids))
    const leftover = [...nodeMap.keys()].filter(id => !claimed.has(id))
    if (leftover.length) subgraphs.push({ title: '', ids: leftover })
  }

  return { direction, nodes: [...nodeMap.values()], subgraphs }
}

function subgraphTitle(value: string): string {
  let title = value.trim()
  const named = title.match(/^(?:[\w-]+\s+)?\[(["']?)(.+)\1\]$/)
  if (named) return named[2].trim()
  return title.replace(/^["']|["']$/g, '').trim()
}

function parseFlowLine(line: string): { nodes: { id: string; label: string }[] } | null {
  const nodes: { id: string; label: string }[] = []
  let i = 0
  const s = line
  const isIdentStart = (ch: string) => /[A-Za-z_]/.test(ch)
  const isIdent = (ch: string) => /[A-Za-z0-9_-]/.test(ch)

  while (i < s.length) {
    while (i < s.length && /\s/.test(s[i])) i++
    if (i >= s.length) break
    if (s.startsWith('-->', i) || s.startsWith('---', i) || s.startsWith('==>', i) || s.startsWith('-.->', i) || s.startsWith('--', i)) {
      if (s.startsWith('-.->', i)) i += 4
      else if (s.startsWith('-->', i) || s.startsWith('==>', i) || s.startsWith('---', i)) i += 3
      else i += 2
      if (s[i] === '|') {
        i++
        while (i < s.length && s[i] !== '|') i++
        if (s[i] === '|') i++
      }
      continue
    }
    if (!isIdentStart(s[i])) return nodes.length ? { nodes } : null
    const start = i
    i++
    while (i < s.length && isIdent(s[i])) i++
    const id = s.slice(start, i)
    let label = id
    const opener = s[i]
    if (opener === '[' || opener === '(' || opener === '{') {
      const closer = opener === '[' ? ']' : opener === '(' ? ')' : '}'
      i++
      if (s[i] === '"' || s[i] === "'") {
        const quote = s[i]
        i++
        const labelStart = i
        while (i < s.length && s[i] !== quote) i++
        label = s.slice(labelStart, i)
        if (s[i] === quote) i++
        if (s[i] === closer) i++
      } else {
        const labelStart = i
        while (i < s.length && s[i] !== closer) i++
        label = s.slice(labelStart, i)
        if (s[i] === closer) i++
      }
    }
    nodes.push({ id, label })
  }
  return nodes.length ? { nodes } : null
}

function toneFromFill(attrs: string): string {
  const match = attrs.match(FILL_RE)
  if (!match) return 'neutral'
  const fill = match[1].toLowerCase()
  if (fill.startsWith('ffe') || fill.startsWith('ef6') || fill.startsWith('ff98') || fill.startsWith('f57')) return 'warm'
  if (fill.startsWith('ffc') || fill.startsWith('c62') || fill.startsWith('e57') || fill.startsWith('ef9') || fill.startsWith('f44')) return 'hot'
  if (fill.startsWith('e3') || fill.startsWith('bb') || fill.startsWith('90') || fill.startsWith('15') || fill.startsWith('19') || fill.startsWith('21') || fill.startsWith('42')) return 'cool'
  return 'neutral'
}

function renderMermaidHTML(graph: { direction: 'td' | 'lr'; nodes: MermaidNode[]; subgraphs: MermaidSubgraph[] }): string {
  const byId = new Map(graph.nodes.map(node => [node.id, node]))
  const titles = graph.subgraphs.map(group => group.title).filter(Boolean)
  const label = escape(titles.join('; ') || 'Diagram')
  const groups = graph.subgraphs.map(group => {
    const title = group.title ? `<p class="content-mermaid-title">${escape(group.title)}</p>` : ''
    const items = group.ids.map(id => {
      const node = byId.get(id)
      if (!node) return ''
      return `<li class="content-mermaid-node content-mermaid-node-${node.tone}">${escape(node.label)}</li>`
    }).join('')
    return `<div class="content-mermaid-subgraph">${title}<ol class="content-mermaid-nodes">${items}</ol></div>`
  }).join('')
  const description = graph.subgraphs.map(group => {
    const labels = group.ids.map(id => byId.get(id)?.label || id)
    const prefix = group.title ? `${escape(group.title)}: ` : ''
    return `<p>${prefix}${labels.map(escape).join(' → ')}.</p>`
  }).join('')
  return `<figure class="content-figure content-diagram content-mermaid"><div class="content-diagram-scroll" role="img" aria-label="${label}"><div class="content-mermaid-graph content-mermaid-${graph.direction}">${groups}</div></div><details><summary>Diagram description</summary><div class="content-media-description">${description}</div></details></figure>`
}

function mermaidCodeBlock(source: string): string {
  const body = source.replace(/\r\n/g, '\n')
  const ended = body.endsWith('\n') ? body : `${body}\n`
  return `<pre><code class="language-mermaid">${escape(ended)}</code></pre>`
}

function escape(value: string): string {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
