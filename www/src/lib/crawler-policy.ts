/**
 * Explicit crawler access posture (SEO.2).
 *
 * Three jobs: training (model corpora), retrieval (search/index), user-fetch
 * (on-demand agent fetches). Our stance is maximum retrievability — every
 * named agent is Allow: /. Change posture by editing this file only.
 */

export type CrawlerJob = 'training' | 'retrieval' | 'user-fetch'

export type CrawlerAgent = {
  /** robots.txt User-agent token */
  agent: string
  job: CrawlerJob
  allow: boolean
  rationale: string
}

/** Paths that must never be indexed (duplicates, errors, thank-you). */
export const DISALLOW_PATHS: string[] = [
  '/404',
  '/*?*', // query-string variants that duplicate canonicals (e.g. ?coupon=)
]

/**
 * Named agents from SEO.2 FR-2. Order: major engines, then AI crawlers by vendor.
 * Adding a bot: one entry with required `job` + `rationale`.
 */
export const CRAWLER_AGENTS: CrawlerAgent[] = [
  // Google
  {
    agent: 'Googlebot',
    job: 'retrieval',
    allow: true,
    rationale: 'Primary web search index for Google organic results.',
  },
  {
    agent: 'Googlebot-Image',
    job: 'retrieval',
    allow: true,
    rationale: 'Image search and multimodal SERP features.',
  },
  {
    agent: 'Google-Extended',
    job: 'training',
    allow: true,
    rationale: 'Gemini / Google AI training and grounding corpus.',
  },
  // Microsoft / Bing / Copilot / ChatGPT retrieval
  {
    agent: 'Bingbot',
    job: 'retrieval',
    allow: true,
    rationale: 'Bing index — also backs ChatGPT Search and Copilot retrieval.',
  },
  {
    agent: 'GPTBot',
    job: 'training',
    allow: true,
    rationale: 'OpenAI model training corpus.',
  },
  {
    agent: 'OAI-SearchBot',
    job: 'retrieval',
    allow: true,
    rationale: 'OpenAI search index for ChatGPT citations.',
  },
  {
    agent: 'ChatGPT-User',
    job: 'user-fetch',
    allow: true,
    rationale: 'Live user-initiated fetches from ChatGPT browsing.',
  },
  // Anthropic
  {
    agent: 'ClaudeBot',
    job: 'training',
    allow: true,
    rationale: 'Anthropic model training corpus.',
  },
  {
    agent: 'Claude-SearchBot',
    job: 'retrieval',
    allow: true,
    rationale: 'Claude search / citation index.',
  },
  {
    agent: 'Claude-User',
    job: 'user-fetch',
    allow: true,
    rationale: 'Live user-initiated fetches from Claude.',
  },
  // Perplexity
  {
    agent: 'PerplexityBot',
    job: 'retrieval',
    allow: true,
    rationale: 'Perplexity answer-engine index.',
  },
  {
    agent: 'Perplexity-User',
    job: 'user-fetch',
    allow: true,
    rationale: 'Live user-initiated fetches from Perplexity.',
  },
  // Apple
  {
    agent: 'Applebot',
    job: 'retrieval',
    allow: true,
    rationale: 'Apple search and Spotlight web results.',
  },
  {
    agent: 'Applebot-Extended',
    job: 'training',
    allow: true,
    rationale: 'Apple Intelligence / foundation model training.',
  },
  // Other major AI + search
  {
    agent: 'Amazonbot',
    job: 'retrieval',
    allow: true,
    rationale: 'Amazon product/knowledge retrieval surfaces.',
  },
  {
    agent: 'Bytespider',
    job: 'training',
    allow: true,
    rationale: 'ByteDance / TikTok training crawler.',
  },
  {
    agent: 'CCBot',
    job: 'training',
    allow: true,
    rationale: 'Common Crawl — feeds many open and commercial models.',
  },
  {
    agent: 'meta-externalagent',
    job: 'training',
    allow: true,
    rationale: 'Meta AI training and retrieval crawler.',
  },
  {
    agent: 'DuckDuckBot',
    job: 'retrieval',
    allow: true,
    rationale: 'DuckDuckGo search index.',
  },
  {
    agent: 'YandexBot',
    job: 'retrieval',
    allow: true,
    rationale: 'Yandex search index.',
  },
  {
    agent: 'cohere-ai',
    job: 'training',
    allow: true,
    rationale: 'Cohere model training corpus.',
  },
  {
    agent: 'Diffbot',
    job: 'retrieval',
    allow: true,
    rationale: 'Structured-web knowledge graphs used by several AI products.',
  },
  {
    agent: 'Timpibot',
    job: 'training',
    allow: true,
    rationale: 'Timpi distributed crawl used by independent AI indexes.',
  },
]

const JOB_LABEL: Record<CrawlerJob, string> = {
  training: 'Training crawlers (model corpora)',
  retrieval: 'Retrieval / search-index crawlers',
  'user-fetch': 'User-initiated live fetch agents',
}

export type RenderRobotsOptions = {
  siteOrigin: string
  /** When true, emit Disallow: / for all agents (staging). */
  disallowAll?: boolean
  agents?: CrawlerAgent[]
  disallowPaths?: string[]
}

/**
 * Render robots.txt from the typed policy.
 * Groups by job with comments; no redundant Allow: /courses lines.
 */
export function renderRobotsTxt(opts: RenderRobotsOptions): string {
  const origin = opts.siteOrigin.replace(/\/$/, '')
  const agents = opts.agents ?? CRAWLER_AGENTS
  const disallowPaths = opts.disallowPaths ?? DISALLOW_PATHS

  if (opts.disallowAll) {
    return [
      '# Staging / non-production — do not index',
      'User-agent: *',
      'Disallow: /',
      '',
      `Sitemap: ${origin}/sitemap.xml`,
      '',
    ].join('\n')
  }

  const lines: string[] = [
    '# Lextures crawler policy — generated from src/lib/crawler-policy.ts (SEO.2)',
    '# Stance: maximum retrievability. Every named agent is Allow: / unless noted.',
    '# Jobs: training (corpora) · retrieval (search indexes) · user-fetch (live agent)',
    '# Do not hand-edit this file; change the TypeScript source and rebuild.',
    '',
  ]

  const jobs: CrawlerJob[] = ['retrieval', 'training', 'user-fetch']
  for (const job of jobs) {
    const group = agents.filter(a => a.job === job)
    if (!group.length) continue
    lines.push(`# --- ${JOB_LABEL[job]} ---`)
    for (const a of group) {
      lines.push(`# ${a.rationale}`)
      lines.push(`User-agent: ${a.agent}`)
      if (a.allow) {
        lines.push('Allow: /')
      } else {
        lines.push('Disallow: /')
      }
      lines.push('')
    }
  }

  lines.push('# --- Default (all other agents) ---')
  lines.push('User-agent: *')
  lines.push('Allow: /')
  for (const p of disallowPaths) {
    lines.push(`Disallow: ${p}`)
  }
  lines.push('')
  lines.push(`Sitemap: ${origin}/sitemap.xml`)
  lines.push('')

  return lines.join('\n')
}

/** Agent names required by SEO.2 FR-2 / AC-1 (for tests and CI). */
export const REQUIRED_CRAWLER_AGENTS = CRAWLER_AGENTS.map(a => a.agent)
