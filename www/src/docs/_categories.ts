import { contentSource } from '../lib/content-source'

export type HelpCategory = {
  id: string
  title: string
  description: string
  order: number
  platformPath: string
}

// Category labels and ordering are supplied by the public content index. The
// related product link is presentation metadata, not article content.
const PLATFORM_PATHS: Record<string, string> = {
  accessibility: '/platform/accessibility',
  accounts: '/security',
  adaptive: '/platform/adaptive-learning',
  assessment: '/platform/assessment',
  grading: '/platform/grading',
  integrations: '/integrations',
  marketplace: '/courses',
  outcomes: '/platform/analytics',
  parents: '/parents',
  'self-hosting': '/get-started',
}

export const HELP_CATEGORIES: HelpCategory[] = contentSource.listCategories().map((category, index) => ({
  id: category.slug,
  title: category.title,
  description: category.description,
  order: category.sortOrder ?? index,
  platformPath: PLATFORM_PATHS[category.slug] || '/platform',
})).sort((a, b) => a.order - b.order)

export const HELP_CATEGORY_IDS = new Set(HELP_CATEGORIES.map(category => category.id))

export function getHelpCategory(id: string): HelpCategory | undefined {
  return HELP_CATEGORIES.find(category => category.id === id)
}
