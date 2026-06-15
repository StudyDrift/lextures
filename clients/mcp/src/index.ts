#!/usr/bin/env node
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js'
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js'
import { z } from 'zod'

const apiUrl = (process.env.LEXTURES_API_URL ?? 'http://localhost:8080').replace(/\/$/, '')
const apiToken = process.env.LEXTURES_API_TOKEN?.trim() ?? ''

async function apiGet(path: string): Promise<unknown> {
  if (!apiToken) {
    throw new Error('LEXTURES_API_TOKEN is required')
  }
  const res = await fetch(`${apiUrl}${path}`, {
    headers: {
      Authorization: `Bearer ${apiToken}`,
      Accept: 'application/json',
    },
  })
  const text = await res.text()
  if (!res.ok) {
    throw new Error(`API ${res.status}: ${text.slice(0, 400)}`)
  }
  return text ? JSON.parse(text) : {}
}

const server = new McpServer({
  name: 'lextures',
  version: '0.1.0',
})

server.tool(
  'list_courses',
  'List courses visible to the authenticated user',
  {
    termId: z.string().uuid().optional().describe('Optional academic term UUID filter'),
  },
  async ({ termId }) => {
    const q = termId ? `?term_id=${encodeURIComponent(termId)}` : ''
    const data = await apiGet(`/api/v1/courses${q}`)
    return {
      content: [{ type: 'text', text: JSON.stringify(data, null, 2) }],
    }
  },
)

server.tool(
  'whoami',
  'Return the authenticated Lextures user profile',
  {},
  async () => {
    const data = await apiGet('/api/v1/me')
    return {
      content: [{ type: 'text', text: JSON.stringify(data, null, 2) }],
    }
  },
)

async function main() {
  const transport = new StdioServerTransport()
  await server.connect(transport)
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
