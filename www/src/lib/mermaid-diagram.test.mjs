import assert from 'node:assert/strict'
import test from 'node:test'
import { parseMermaidGraph, renderMermaidBlock } from './mermaid-diagram.ts'

const source = `graph TD
    subgraph "Traditional Linear Approach"
        T1[1. Remember] --> T2[2. Understand] --> T3[3. Apply]
        style T1 fill:#e3f2fd,stroke:#1565c0
        style T3 fill:#ffe0b2,stroke:#ef6c00
    end
    subgraph "AI-Inverted Approach"
        I3[3. Create] --> I1[1. Remember]
        style I3 fill:#ffcdd2,stroke:#c62828
    end
`

test('parses mermaid flowcharts into labeled subgraphs', () => {
  const graph = parseMermaidGraph(source)
  assert.ok(graph)
  assert.equal(graph.direction, 'td')
  assert.equal(graph.subgraphs.length, 2)
  assert.equal(graph.subgraphs[0].title, 'Traditional Linear Approach')
  assert.deepEqual(graph.subgraphs[0].ids, ['T1', 'T2', 'T3'])
  assert.equal(graph.nodes.find(node => node.id === 'T1')?.tone, 'cool')
  assert.equal(graph.nodes.find(node => node.id === 'T3')?.tone, 'warm')
  assert.equal(graph.nodes.find(node => node.id === 'I3')?.tone, 'hot')
})

test('renders an accessible HTML diagram instead of a code fence', () => {
  const html = renderMermaidBlock(source)
  assert.match(html, /<figure class="content-figure content-diagram content-mermaid">/)
  assert.match(html, /aria-label="Traditional Linear Approach; AI-Inverted Approach"/)
  assert.match(html, /1\. Remember/)
  assert.doesNotMatch(html, /<pre><code class="language-mermaid">/)
})

test('leaves unsupported mermaid dialects as source', () => {
  const html = renderMermaidBlock('sequenceDiagram\n    Alice->>Bob: Hello\n')
  assert.match(html, /<pre><code class="language-mermaid">/)
  assert.match(html, /Alice-&gt;&gt;Bob: Hello/)
})
